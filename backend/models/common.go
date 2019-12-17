package models

import (
	"fmt"
	"strings"
	"time"

	"kubecloud/backend/service"

	"github.com/astaxie/beego/orm"
	"github.com/go-sql-driver/mysql"
)

// NOTE: should perfer AddonsUnix, Addons will be deprecated in future

// AddonsUnix ...
type AddonsUnix struct {
	Deleted   int8   `orm:"column(deleted)" json:"deleted"`
	CreatedAt int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt int64  `orm:"column(updated_at)" json:"updated_at"`
	DeletedAt *int64 `orm:"column(deleted_at);null" json:"deleted_at"`
}

// NewAddonsUnix ...
func NewAddonsUnix() AddonsUnix {
	now := time.Now().Unix()
	return AddonsUnix{
		Deleted:   0,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
	}
}

// MarkUpdated ...
func (a *AddonsUnix) MarkUpdated() {
	now := time.Now().Unix()
	a.UpdatedAt = now
}

// MarkDeleted ...
func (a *AddonsUnix) MarkDeleted() {
	now := time.Now().Unix()
	a.DeletedAt = &now
	a.Deleted = 1
}

type Addons struct {
	Deleted  int8      `orm:"column(deleted)" json:"deleted"`
	CreateAt time.Time `orm:"column(create_at)" json:"create_at"`
	UpdateAt time.Time `orm:"column(update_at)" json:"update_at"`
	DeleteAt time.Time `orm:"column(delete_at);null" json:"delete_at"`
}

func NewAddonsWithCreationTimestamp(timestamp time.Time) Addons {
	timeNow := timestamp
	timeDel, _ := time.Parse("2006-01-02 15:04:05", time.Unix(0, 0).Local().Format("2006-01-02 15:04:05"))
	return Addons{
		Deleted:  0,
		CreateAt: timeNow,
		UpdateAt: timeNow,
		DeleteAt: timeDel,
	}
}

func NewAddons() Addons {
	timeNow, _ := time.Parse("2006-01-02 15:04:05", time.Now().Local().Format("2006-01-02 15:04:05"))
	timeDel, _ := time.Parse("2006-01-02 15:04:05", time.Unix(0, 0).Local().Format("2006-01-02 15:04:05"))
	return Addons{
		Deleted:  0,
		CreateAt: timeNow,
		UpdateAt: timeNow,
		DeleteAt: timeDel,
	}
}

func (ons Addons) UpdateAddons() Addons {
	ons.CreateAt, _ = time.Parse("2006-01-02 15:04:05", ons.CreateAt.Format("2006-01-02 15:04:05"))
	ons.UpdateAt, _ = time.Parse("2006-01-02 15:04:05", time.Now().Local().Format("2006-01-02 15:04:05"))
	return ons
}

func (ons Addons) FormatAddons() Addons {
	ons.CreateAt, _ = time.Parse("2006-01-02 15:04:05", ons.CreateAt.Format("2006-01-02 15:04:05"))
	ons.UpdateAt, _ = time.Parse("2006-01-02 15:04:05", ons.UpdateAt.Format("2006-01-02 15:04:05"))
	return ons
}

type HardAddons struct {
	CreateAt time.Time `orm:"column(create_at)" json:"create_at"`
	UpdateAt time.Time `orm:"column(update_at)" json:"update_at"`
}

func NewHardAddons() HardAddons {
	timeNow, _ := time.Parse("2006-01-02 15:04:05", time.Now().Local().Format("2006-01-02 15:04:05"))
	return HardAddons{
		CreateAt: timeNow,
		UpdateAt: timeNow,
	}
}

var (
	dbName     string
	tableNames []string
)

func initOrm() {
	config := service.GetAppConfig()
	DatabaseDebug, _ := config.Bool("DB::databaseDebug")
	DefaultRowsLimit, _ := config.Int("DB::defaultRowsLimit")
	DatabaseUrl := config.String("DB::databaseUrl")

	if cfg, err := mysql.ParseDSN(DatabaseUrl); err == nil {
		dbName = cfg.DBName
	}

	orm.Debug = DatabaseDebug
	if DefaultRowsLimit != 0 {
		orm.DefaultRowsLimit = DefaultRowsLimit
	}

	if err := orm.RegisterDriver("mysql", orm.DRMySQL); err != nil {
		panic(fmt.Sprintf(`failed to register driver, error: "%s"`, err.Error()))
	}
	if err := orm.RegisterDataBase("default", "mysql", DatabaseUrl); err != nil {
		panic(fmt.Sprintf(`failed to register database, error: "%s", url: "%s"`, err.Error(), DatabaseUrl))
	}
	registerModel := func(models ...interface{}) {
		tableNames = make([]string, len(models))
		for i, model := range models {
			obj := model.(interface {
				TableName() string
			})
			tableNames[i] = obj.TableName()
		}
		orm.RegisterModel(models...)
	}
	registerModel(
		new(ZcloudCluster),
		new(ZcloudNode),
		new(ZcloudHarbor),
		new(ZcloudHarborProject),
		new(ZcloudHarborRepository),
		new(K8sSecret),
		new(ZcloudEvent),
		new(ZcloudTemplate),
		new(ZcloudApplication),
		new(ZcloudVersion),
		new(K8sIngress),
		new(K8sIngressRule),
		new(K8sService),
		new(K8sServicePort),
		new(K8sEndpoint),
		new(K8sEndpointAddress),
		new(K8sNamespace),
		new(ZcloudRepositoryTag),
		new(ZcloudClusterDomainSuffix),
	)
}

func Init() {
	initOrm()
	orm.RunSyncdb("default", false, true)
}

func InitMock() {
	initOrm()

	// check database name and clear all tables
	if !strings.Contains(dbName, "test") {
		panic(fmt.Sprintf(`invalid database: "%v"`, dbName))
	}

	orm.RunSyncdb("default", false, true)

	ormer := orm.NewOrm()
	for _, name := range tableNames {
		if _, err := ormer.Raw(fmt.Sprintf("truncate table %v", name)).Exec(); err != nil {
			panic(err)
		}
	}
}
