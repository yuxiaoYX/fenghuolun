package owner

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"fenghuolun/internal/coded"
	"fenghuolun/internal/neta"
	"fenghuolun/internal/store"
)

const minPassword = 8

var (
	phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
	codePattern  = regexp.MustCompile(`^\d{4,8}$`)
)

type attemptBook struct {
	mu sync.Mutex
	at map[string][]time.Time
}

func (b *attemptBook) count(key string, window time.Duration) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.at == nil {
		return 0
	}
	cut := time.Now().Add(-window)
	kept := make([]time.Time, 0, len(b.at[key]))
	for _, t := range b.at[key] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	b.at[key] = kept
	return len(kept)
}

func (b *attemptBook) add(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.at == nil {
		b.at = map[string][]time.Time{}
	}
	b.at[key] = append(b.at[key], time.Now())
}

type Enter struct {
	Session string
	Bound   bool
	Binding *store.Binding
}

func (s *Service) AccountSession(session string) bool {
	if session == "" || s.Store == nil {
		return false
	}
	id, err := s.Store.AccountBySession(session)
	return err == nil && id != ""
}

func (s *Service) SendLoginCode(phone string) error {
	phone = normalizePhone(phone)
	if !phonePattern.MatchString(phone) {
		return coded.New(http.StatusBadRequest, "invalid_request", "请填写 11 位手机号")
	}
	if s.limits.count(phone+"#gap", time.Minute) >= 1 {
		return coded.New(http.StatusBadRequest, "invalid_request", "请稍后再获取验证码")
	}
	if s.limits.count(phone+"#send", time.Hour) >= 5 {
		return coded.New(http.StatusBadRequest, "invalid_request", "获取次数过多，请稍后再试")
	}
	s.limits.add(phone + "#send")
	if err := s.Client.SendLoginCode(phone); err != nil {
		return mapLoginUpstream(err)
	}
	s.limits.add(phone + "#gap")
	return nil
}

func (s *Service) AccountPhone(session string) (string, error) {
	accountID, err := s.Store.AccountBySession(session)
	if err != nil {
		return "", err
	}
	if accountID == "" {
		return "", coded.New(http.StatusUnauthorized, "unauthorized", "请先登录")
	}
	phone, err := s.Store.OwnerPhoneByID(accountID)
	if err != nil || !phonePattern.MatchString(phone) {
		return "", coded.New(http.StatusBadRequest, "invalid_request", "风火轮账号手机号无效")
	}
	return phone, nil
}

func (s *Service) ChangePhone(session, phone, password string) error {
	accountID, err := s.Store.AccountBySession(session)
	if err != nil {
		return err
	}
	if accountID == "" {
		return coded.New(http.StatusUnauthorized, "unauthorized", "请先登录")
	}
	phone = normalizePhone(phone)
	if !phonePattern.MatchString(phone) {
		return coded.New(http.StatusBadRequest, "invalid_request", "请填写 11 位手机号")
	}
	if err := checkPassword(password); err != nil {
		return err
	}
	acc, err := s.Store.AccountByID(accountID)
	if err != nil || acc == nil || bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)) != nil {
		return coded.New(http.StatusUnauthorized, "unauthorized", "密码不正确")
	}
	other, err := s.Store.OwnerByPhone(phone)
	if err != nil {
		return err
	}
	if other != nil && other.Id != accountID {
		return coded.New(http.StatusConflict, "conflict", "手机号已注册，请换一个手机号")
	}
	return s.Store.UpdateOwnerPhone(accountID, phone)
}

func (s *Service) Register(phone, password, code, refreshToken string) (*Enter, error) {
	phone = normalizePhone(phone)
	code = strings.TrimSpace(code)
	refreshToken = strings.TrimSpace(refreshToken)
	if !phonePattern.MatchString(phone) {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "请填写 11 位手机号")
	}
	if err := checkPassword(password); err != nil {
		return nil, err
	}
	if code != "" && refreshToken != "" {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "验证码和 refresh_token 只能选一种")
	}
	if code == "" && refreshToken == "" {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "请获取验证码，或填写 refresh_token")
	}
	acc, err := s.Store.OwnerByPhone(phone)
	if err != nil {
		return nil, err
	}
	if acc != nil {
		return nil, coded.New(http.StatusConflict, "conflict", "手机号已注册，请登录")
	}
	var pair neta.TokenPair
	if code != "" {
		if !codePattern.MatchString(code) {
			return nil, coded.New(http.StatusBadRequest, "invalid_request", "请填写短信验证码")
		}
		if !s.limits.try(phone+"#smslogin", 10*time.Minute, 5) {
			return nil, coded.New(http.StatusBadRequest, "invalid_request", "验证太频繁，请稍后再试")
		}
		pair, err = s.Client.LoginBySMS(phone, code)
		if err != nil {
			return nil, mapLoginUpstream(err)
		}
	} else {
		if len(refreshToken) < 8 {
			return nil, coded.New(http.StatusBadRequest, "invalid_request", "请重新填写 refresh_token")
		}
		pair, err = s.Client.Refresh(refreshToken)
		if err != nil {
			return nil, err
		}
	}
	live, err := s.pullLive(pair)
	if err != nil {
		return nil, err
	}
	sum, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id, err := s.Store.CreateOwnerAccount(phone, string(sum))
	if err != nil {
		return nil, err
	}
	b, err := s.storeOfficial(pair, id, "", live)
	if err != nil {
		_ = s.Store.DeleteOwnerAccount(id)
		return nil, err
	}
	return &Enter{Session: b.Session, Bound: true, Binding: b}, nil
}

func (s *Service) Login(phone, password string) (*Enter, error) {
	phone = normalizePhone(phone)
	if !phonePattern.MatchString(phone) {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "请填写 11 位手机号")
	}
	if err := checkPassword(password); err != nil {
		return nil, err
	}
	if s.limits.count(phone+"#pwfail", 15*time.Minute) >= 8 {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "尝试过多，请稍后再试")
	}
	acc, err := s.Store.OwnerByPhone(phone)
	if err != nil {
		return nil, err
	}
	if acc == nil || bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)) != nil {
		s.limits.add(phone + "#pwfail")
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "手机号或密码不正确")
	}
	if err := s.rejectDisabled(acc.BindingId); err != nil {
		return nil, err
	}
	if cur := s.Store.GetByID(acc.BindingId); cur != nil && cur.RefreshToken != "" {
		session, err := s.Store.IssueSession(cur.ID, acc.Id)
		if err != nil {
			return nil, err
		}
		got := s.Store.Get(session)
		if got == nil {
			return nil, coded.New(http.StatusBadGateway, "upstream", "登录会话没有建立")
		}
		return &Enter{Session: session, Bound: true, Binding: got}, nil
	}
	session, err := s.Store.IssueAccountSession(acc.Id)
	if err != nil {
		return nil, err
	}
	return &Enter{Session: session, Bound: false}, nil
}

func (s *Service) BindBySMS(session, code string) (*store.Binding, error) {
	code = strings.TrimSpace(code)
	if !codePattern.MatchString(code) {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "请填写短信验证码")
	}
	accountID, err := s.Store.AccountBySession(session)
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return nil, coded.New(http.StatusUnauthorized, "unauthorized", "请先登录")
	}
	phone, err := s.Store.OwnerPhoneByID(accountID)
	if err != nil || !phonePattern.MatchString(phone) {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "风火轮账号手机号无效")
	}
	if !s.limits.try(phone+"#smslogin", 10*time.Minute, 5) {
		return nil, coded.New(http.StatusBadRequest, "invalid_request", "验证太频繁，请稍后再试")
	}
	pair, err := s.Client.LoginBySMS(phone, code)
	if err != nil {
		return nil, mapLoginUpstream(err)
	}
	return s.adoptOfficial(pair, accountID)
}

func (s *Service) BindToken(session, refreshToken string) (*store.Binding, error) {
	if s.Get(session) != nil {
		return s.Rebind(session, refreshToken)
	}
	accountID, err := s.Store.AccountBySession(session)
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return s.Bind(refreshToken)
	}
	b, err := s.Bind(refreshToken)
	if err != nil {
		return nil, err
	}
	if err := s.attachAccount(accountID, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) adoptOfficial(pair neta.TokenPair, accountID string) (*store.Binding, error) {
	b, err := s.pullLive(pair)
	if err != nil {
		return nil, err
	}
	return s.storeOfficial(pair, accountID, "", b)
}

func (s *Service) attachAccount(accountID string, b *store.Binding) error {
	if accountID == "" || b == nil {
		return nil
	}
	if err := s.Store.SetAccountBinding(accountID, b.ID); err != nil {
		return err
	}
	return s.Store.StampSessionAccount(b.Session, accountID)
}

func (b *attemptBook) try(key string, window time.Duration, max int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.at == nil {
		b.at = map[string][]time.Time{}
	}
	now := time.Now()
	cut := now.Add(-window)
	kept := make([]time.Time, 0, len(b.at[key])+1)
	for _, t := range b.at[key] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= max {
		b.at[key] = kept
		return false
	}
	b.at[key] = append(kept, now)
	return true
}

func (s *Service) rejectDisabled(bindingID string) error {
	if bindingID == "" {
		return nil
	}
	cur := s.Store.GetByIDAny(bindingID)
	if cur != nil && cur.Disabled {
		return coded.New(http.StatusForbidden, "forbidden", "该车已被管理员停用")
	}
	return nil
}

func mapLoginUpstream(err error) error {
	var api *neta.APIError
	if errors.As(err, &api) {
		msg := strings.TrimSpace(api.Description)
		if msg == "" || len([]rune(msg)) > 40 {
			msg = "官方登录没有通过"
		}
		switch {
		case errors.Is(err, neta.ErrSMSRejected), errors.Is(err, neta.ErrTokenInvalid):
			return coded.New(http.StatusUnauthorized, "sms_invalid", msg)
		case errors.Is(err, neta.ErrSignRequired):
			return coded.New(http.StatusBadGateway, "sign_required", "官方接口："+msg)
		default:
			return coded.New(http.StatusBadGateway, "upstream", "官方接口："+msg)
		}
	}
	switch {
	case errors.Is(err, neta.ErrSMSRejected), errors.Is(err, neta.ErrTokenInvalid):
		return coded.New(http.StatusUnauthorized, "sms_invalid", "验证码不正确或已失效")
	case errors.Is(err, neta.ErrSignRequired):
		return coded.New(http.StatusBadGateway, "sign_required", "官方登录要求签名，算法尚未验证。请改用手填 refresh_token")
	default:
		return err
	}
}

func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if strings.HasPrefix(s, "86") && len(s) == 13 {
		s = s[2:]
	}
	return s
}

func checkPassword(password string) error {
	if len(password) < minPassword || len(password) > 72 {
		return coded.New(http.StatusBadRequest, "invalid_request", "密码至少 8 位")
	}
	return nil
}
