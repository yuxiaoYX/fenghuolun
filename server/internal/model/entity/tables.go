package entity

type Binding struct {
	Id            string
	VinMasked     string
	VinCipher     []byte
	ModelCode     string
	ModelName     string
	Nickname      string
	Trim          string
	IsExtender    int
	RefreshCipher []byte
	AccessCipher  []byte
	RefreshHint   string
	Disabled      int
	Lng           *float64
	Lat           *float64
	LocReportedAt *int64
	CreatedAt     int64
	UpdatedAt     int64
	DeletedAt     int64
}

type OwnerSession struct {
	TokenHash string
	BindingId string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
}

type Snapshot struct {
	Id        int64
	BindingId string
	FetchedAt int64
	Payload   string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
}

type Energy struct {
	BindingId string
	Payload   string
	FetchedAt int64
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
}

type SyncJob struct {
	Id            string
	BindingId     string
	Kind          string
	Status        string
	ErrorPublic   string
	ErrorInternal string
	StartedAt     int64
	FinishedAt    int64
	CreatedAt     int64
	UpdatedAt     int64
	DeletedAt     int64
}

type FillEvent struct {
	Id          string
	BindingId   string
	Kind        string
	Source      string
	Status      string
	FromFetched int64
	ToFetched   int64
	StartedAt   int64
	FinishedAt  int64
	OdoStart    *float64
	OdoEnd      *float64
	SocStart    *float64
	SocEnd      *float64
	FuelStart   *float64
	FuelEnd     *float64
	EnergyKwh   *float64
	Liters      *float64
	PaidCny     *float64
	Note        string
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   int64
}

type AppSetting struct {
	Key       string
	Value     string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
}

type AdminUser struct {
	Username     string
	PasswordHash string
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    int64
}

type AdminSession struct {
	TokenHash string
	Username  string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
}
