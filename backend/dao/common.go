package dao

import (
	"sync"

	"github.com/astaxie/beego/orm"
)

var globalOrm orm.Ormer
var once sync.Once

// GetOrmer get ormer singleton. Pitfall: each transaction requires a separate orm
func GetOrmer() orm.Ormer {
	once.Do(func() {
		globalOrm = orm.NewOrm()
	})
	return globalOrm
}
