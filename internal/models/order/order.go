// Package order — доменная модель заказа.
//
// Здесь живут типы предметной области и правила, которые к ним относятся.
// Пакет ничего не знает про HTTP, JSON и базу данных: он не импортирует
// net/http и не содержит тегов сериализации. Это проверяется на приёмке.
package order

import (
	"errors"
	"fmt"
	"time"
)

// 00 - типы заказа и правила домена
var ErrValidation error = errors.New("нарушены правила предметной области")
var ErrNotFound error = errors.New("не найден такой заказ в repository")

type Item struct {
	Sku         string
	Qty         int
	Price_cents int
}

type Order struct {
	Id          string
	Customer_id string
	Status      string
	Items       []Item
	Total_cents int
	Created_at  time.Time
}

// TODO delete // - for my debuging
func (o *Order) Println() {
	fmt.Println("id: ", o.Id)
	fmt.Println("consumer_id: ", o.Customer_id)
	fmt.Println("items: ", o.Items)
	fmt.Println("total_cents: ", o.Total_cents)
	fmt.Println("created_at: ", o.Created_at)
}
