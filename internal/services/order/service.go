// Package order — сервисный слой заказов: сценарии поверх домена и хранилища.
//
// Здесь решается, что вообще происходит: провалидировать, посчитать сумму,
// сходить в хранилище, вернуть доменную ошибку. Контроллер после этого
// занимается только переводом в HTTP.
package order

import (
	"time"

	OrderModel "github.com/gulmix/orders/internal/models/order"
	Repository "github.com/gulmix/orders/internal/repositories/order"
)

// 00 - сценарии создания и получения

func MakeOrder(customer_id string, items []OrderModel.Item) (OrderModel.Order, error) {

	ord := OrderModel.Order{Customer_id: customer_id, Items: items}

	if err := ValidateOrder(ord); err != nil {
		return ord, OrderModel.ErrValidation
	}

	ord.Status = "created"

	ord.Total_cents = 0
	for _, item := range ord.Items {
		ord.Total_cents += item.Qty * item.Price_cents
	}

	ord.Created_at = time.Now()

	if err := Repository.SaveOrder(&ord); err != nil {
		return ord, err
	}

	// ord.Println() // FIX delete this line later

	return ord, nil
}

func ValidateOrder(ord OrderModel.Order) error {
	if ord.Customer_id == "" {
		return OrderModel.ErrValidation
	}
	if len(ord.Items) == 0 {
		return OrderModel.ErrValidation
	}
	for _, item := range ord.Items {
		if item.Sku == "" || item.Qty <= 0 || item.Price_cents < 0 {
			return OrderModel.ErrValidation
		}
	}
	return nil
}

func GetOrder(id string) (ord OrderModel.Order, err error) {
	ord, err = Repository.GetOrder(id)
	if err != nil {
		return ord, err
	}
	return ord, nil
}
