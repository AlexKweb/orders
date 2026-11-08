// Package order — хранилище заказов.
//
// В Модуле 0 это карта в памяти. В Модуле 5 на её место встанет Postgres,
// и если границы проведены правильно, менять придётся только этот пакет.
package order

//00-хранилище в памяти

import (
	"strconv"
	"sync"

	OrderModel "github.com/gulmix/orders/internal/models/order"
)

var Repository = make(map[string]OrderModel.Order, 0)
var mtx = sync.RWMutex{}

func SaveOrder(ord *OrderModel.Order) error {

	mtx.Lock()
	ord.Id = "ord_" + strconv.Itoa(len(Repository)+1) //FIX стоит заменить потом на UUID и по возможности перенести в сервис
	Repository[ord.Id] = *ord
	mtx.Unlock()
	return nil
}

func GetOrder(id string) (OrderModel.Order, error) {
	mtx.RLock()
	ord, ok := Repository[id]
	mtx.RUnlock()
	if !ok {
		return ord, OrderModel.ErrNotFound
	}
	return ord, nil
}
