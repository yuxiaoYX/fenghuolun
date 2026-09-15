package do

import "github.com/gogf/gf/v2/frame/g"

// 与 migrate 表结构对齐的 DO。测试用 gdb.New 隔离库，不走 g.DB 单例，故未接 gf gen dao。
// created_at / updated_at / deleted_at 只留给 gdb 自动维护，Data() 里不要赋值。

type Binding struct {
	g.Meta        `orm:"table:binding, do:true"`
	Id            interface{}
	VinMasked     interface{}
	VinCipher     interface{}
	ModelCode     interface{}
	ModelName     interface{}
	Nickname      interface{}
	Trim          interface{}
	IsExtender    interface{}
	RefreshCipher interface{}
	AccessCipher  interface{}
	RefreshHint   interface{}
	Disabled      interface{}
	Lng           interface{}
	Lat           interface{}
	LocReportedAt interface{}
	CreatedAt     interface{}
	UpdatedAt     interface{}
	DeletedAt     interface{}
}

type OwnerSession struct {
	g.Meta    `orm:"table:owner_session, do:true"`
	TokenHash interface{}
	BindingId interface{}
	CreatedAt interface{}
	UpdatedAt interface{}
	DeletedAt interface{}
}

type Snapshot struct {
	g.Meta    `orm:"table:snapshot, do:true"`
	Id        interface{}
	BindingId interface{}
	FetchedAt interface{}
	Payload   interface{}
	CreatedAt interface{}
	UpdatedAt interface{}
	DeletedAt interface{}
}

type Energy struct {
	g.Meta    `orm:"table:energy, do:true"`
	BindingId interface{}
	Payload   interface{}
	FetchedAt interface{}
	CreatedAt interface{}
	UpdatedAt interface{}
	DeletedAt interface{}
}

type SyncJob struct {
	g.Meta      `orm:"table:sync_job, do:true"`
	BindingId   interface{}
	Status      interface{}
	ErrorPublic interface{}
	SyncedAt    interface{}
	CreatedAt   interface{}
	UpdatedAt   interface{}
	DeletedAt   interface{}
}

type AdminUser struct {
	g.Meta       `orm:"table:admin_user, do:true"`
	Username     interface{}
	PasswordHash interface{}
	CreatedAt    interface{}
	UpdatedAt    interface{}
	DeletedAt    interface{}
}

type AdminSession struct {
	g.Meta    `orm:"table:admin_session, do:true"`
	TokenHash interface{}
	CreatedAt interface{}
	UpdatedAt interface{}
	DeletedAt interface{}
}
