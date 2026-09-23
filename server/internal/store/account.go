package store

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"fenghuolun/internal/crypto"
	"fenghuolun/internal/model/do"
	"fenghuolun/internal/model/entity"
)

func (s *SQLite) phoneKey(phone string) string {
	mac := hmac.New(sha256.New, s.kek)
	_, _ = mac.Write([]byte(phone))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *SQLite) OwnerByPhone(phone string) (*entity.OwnerAccount, error) {
	if phone == "" {
		return nil, nil
	}
	ctx := s.ctx()
	m := s.db.Model("owner_account").Ctx(ctx).Where("phone_hash", s.phoneKey(phone))
	n, err := m.Count()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	var row entity.OwnerAccount
	if err := s.db.Model("owner_account").Ctx(ctx).Where("phone_hash", s.phoneKey(phone)).Scan(&row); err != nil {
		return nil, err
	}
	if row.Id == "" {
		return nil, nil
	}
	return &row, nil
}

func (s *SQLite) SaveOwnerAccount(phone, passwordHash, bindingID string, create bool) error {
	if phone == "" || bindingID == "" {
		return fmt.Errorf("invalid_request: phone")
	}
	cipher, err := crypto.Seal(s.kek, phone)
	if err != nil {
		return err
	}
	ctx := s.ctx()
	key := s.phoneKey(phone)
	existing, err := s.OwnerByPhone(phone)
	if err != nil {
		return err
	}
	if existing == nil {
		if !create || passwordHash == "" {
			return fmt.Errorf("invalid_request: phone")
		}
		_, err = s.db.Model("owner_account").Ctx(ctx).Data(do.OwnerAccount{
			Id:           NewSessionID(),
			PhoneHash:    key,
			PhoneCipher:  cipher,
			PasswordHash: passwordHash,
			BindingId:    bindingID,
		}).Insert()
		return err
	}
	_, err = s.db.Model("owner_account").Ctx(ctx).Where("id", existing.Id).Data(do.OwnerAccount{
		PhoneCipher: cipher,
		BindingId:   bindingID,
	}).Update()
	return err
}

func (s *SQLite) IssueSession(bindingID, accountID string) (string, error) {
	if bindingID == "" {
		return "", fmt.Errorf("invalid_request: missing binding id")
	}
	session := NewSessionID()
	_, err := s.db.Model("owner_session").Ctx(s.ctx()).Data(map[string]any{
		"token_hash": hashToken(session),
		"binding_id": bindingID,
		"account_id": accountID,
	}).Insert()
	if err != nil {
		return "", err
	}
	return session, nil
}

func (s *SQLite) IssueAccountSession(accountID string) (string, error) {
	if accountID == "" {
		return "", fmt.Errorf("invalid_request: phone")
	}
	session := NewSessionID()
	_, err := s.db.Model("owner_session").Ctx(s.ctx()).Data(map[string]any{
		"token_hash": hashToken(session),
		"binding_id": "",
		"account_id": accountID,
	}).Insert()
	if err != nil {
		return "", err
	}
	return session, nil
}

func (s *SQLite) AccountBySession(session string) (string, error) {
	if session == "" {
		return "", nil
	}
	v, err := s.db.Model("owner_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Value("account_id")
	if err != nil || v.IsEmpty() {
		return "", err
	}
	return v.String(), nil
}

func (s *SQLite) CreateOwnerAccount(phone, passwordHash string) (string, error) {
	if phone == "" || passwordHash == "" {
		return "", fmt.Errorf("invalid_request: phone")
	}
	cipher, err := crypto.Seal(s.kek, phone)
	if err != nil {
		return "", err
	}
	id := NewSessionID()
	_, err = s.db.Model("owner_account").Ctx(s.ctx()).Data(map[string]any{
		"id":            id,
		"phone_hash":    s.phoneKey(phone),
		"phone_cipher":  cipher,
		"password_hash": passwordHash,
		"binding_id":    "",
	}).Insert()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *SQLite) AccountByID(id string) (*entity.OwnerAccount, error) {
	if id == "" {
		return nil, nil
	}
	n, err := s.db.Model("owner_account").Ctx(s.ctx()).Where("id", id).Count()
	if err != nil || n == 0 {
		return nil, err
	}
	var row entity.OwnerAccount
	if err := s.db.Model("owner_account").Ctx(s.ctx()).Where("id", id).Scan(&row); err != nil {
		return nil, err
	}
	if row.Id == "" {
		return nil, nil
	}
	return &row, nil
}

func (s *SQLite) OwnerPhoneByID(id string) (string, error) {
	row, err := s.AccountByID(id)
	if err != nil || row == nil {
		return "", err
	}
	return crypto.Open(s.kek, row.PhoneCipher)
}

func (s *SQLite) UpdateOwnerPhone(id, phone string) error {
	if id == "" || phone == "" {
		return fmt.Errorf("invalid_request: phone")
	}
	cipher, err := crypto.Seal(s.kek, phone)
	if err != nil {
		return err
	}
	_, err = s.db.Model("owner_account").Ctx(s.ctx()).Where("id", id).Data(do.OwnerAccount{
		PhoneHash:   s.phoneKey(phone),
		PhoneCipher: cipher,
	}).Update()
	return err
}

func (s *SQLite) SetAccountBinding(accountID, bindingID string) error {
	if accountID == "" || bindingID == "" {
		return fmt.Errorf("invalid_request: phone")
	}
	_, err := s.db.Model("owner_account").Ctx(s.ctx()).Where("id", accountID).Data(do.OwnerAccount{
		BindingId: bindingID,
	}).Update()
	return err
}

func (s *SQLite) StampSessionAccount(session, accountID string) error {
	if session == "" || accountID == "" {
		return nil
	}
	_, err := s.db.Model("owner_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Data(map[string]any{
		"account_id": accountID,
	}).Update()
	return err
}

func (s *SQLite) RevokeSession(session string) error {
	if session == "" {
		return nil
	}
	_, err := s.db.Model("owner_session").Ctx(s.ctx()).Where("token_hash", hashToken(session)).Delete()
	return err
}

func (s *SQLite) ClearAccountBinding(accountID, bindingID string) error {
	if accountID == "" {
		return nil
	}
	m := s.db.Model("owner_account").Ctx(s.ctx()).Where("id", accountID)
	if bindingID != "" {
		m = m.Where("binding_id", bindingID)
	}
	_, err := m.Data(do.OwnerAccount{BindingId: ""}).Update()
	return err
}

func (s *SQLite) DeleteOwnerAccount(accountID string) error {
	if accountID == "" {
		return nil
	}
	_, err := s.db.Model("owner_account").Ctx(s.ctx()).Unscoped().Where("id", accountID).Delete()
	return err
}

func MaskPhone(phone string) string {
	if len(phone) < 7 {
		return "—"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}
