// Package order — HTTP-контроллер заказов: разбор запроса, вызов сервиса,
// перевод доменных ошибок в коды ответа.
//
// Единый формат ответа и коды ошибок — в соседнем пакете response.
package order

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gulmix/orders/internal/controllers/http/response"
	OrderModel "github.com/gulmix/orders/internal/models/order"
	OrderServices "github.com/gulmix/orders/internal/services/order"
)

// Controller держит зависимости хендлеров пробы. Их пока нет.
type Controller struct{}

// NewController создаёт контроллер.
func NewController() *Controller {
	return &Controller{}
}

// 00 - хендлеры, разбор запроса, коды
var ErrJsonDecode error = errors.New("не удалось разобрать JSON")

type BodyItem struct {
	Sku         string `json:"sku"`
	Qty         int    `json:"qty"`
	Price_cents int    `json:"price_cents"`
}

type OrderBody struct {
	Id          string     `json:"id"`
	Customer_id string     `json:"customer_id"`
	Status      string     `json:"status"`
	Items       []BodyItem `json:"items"`
	Total_cents int        `json:"total_cents"`
	Created_at  string     `json:"created_at"`
}

func (c *Controller) OrderPost(w http.ResponseWriter, r *http.Request) {

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeDomainError(w, r, ErrJsonDecode)
		return
	}

	ordRe := OrderBody{}

	tmpJson := json.NewDecoder(r.Body)
	tmpJson.DisallowUnknownFields()
	if err := tmpJson.Decode(&ordRe); err != nil {
		writeDomainError(w, r, ErrJsonDecode)
		return
	}

	l := len(ordRe.Items)
	items := make([]OrderModel.Item, l)
	for i := range l {
		items[i].Sku = ordRe.Items[i].Sku
		items[i].Qty = ordRe.Items[i].Qty
		items[i].Price_cents = ordRe.Items[i].Price_cents
	}

	ord, err := OrderServices.MakeOrder(ordRe.Customer_id, items)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	ordRe.Id = ord.Id
	ordRe.Status = ord.Status
	ordRe.Total_cents = ord.Total_cents
	ordRe.Created_at = ord.Created_at.Format(time.RFC3339)

	w.Header().Set("Location", "/orders/"+ord.Id)
	response.JSON(w, r, http.StatusCreated, &ordRe)
}

func (c *Controller) OrderGet(w http.ResponseWriter, r *http.Request) {

	id := r.PathValue("id")

	ord, err := OrderServices.GetOrder(id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	ordRe := OrderBody{}

	ordRe.Id = ord.Id
	ordRe.Customer_id = ord.Customer_id
	ordRe.Status = ord.Status
	ordRe.Items = make([]BodyItem, len(ord.Items))
	for i := range ord.Items {
		ordRe.Items[i].Sku = ord.Items[i].Sku
		ordRe.Items[i].Qty = ord.Items[i].Qty
		ordRe.Items[i].Price_cents = ord.Items[i].Price_cents
	}
	ordRe.Total_cents = ord.Total_cents
	ordRe.Created_at = ord.Created_at.Format(time.RFC3339)

	w.Header().Set("Location", "/orders/"+ord.Id)
	response.JSON(w, r, http.StatusOK, &ordRe)
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrJsonDecode):
		response.Error(w, r, http.StatusBadRequest, response.CodeBadRequest, "не удалось разобрать JSON")
	case errors.Is(err, OrderModel.ErrValidation):
		response.Error(w, r, http.StatusUnprocessableEntity, response.CodeValidation, "нарушены правила предметной области")
	case errors.Is(err, OrderModel.ErrNotFound):
		response.Error(w, r, http.StatusNotFound, response.CodeNotFound, "заказ не найден")
	default:
		slog.ErrorContext(r.Context(), "необработанная ошибка", slog.Any("error", err))
		response.Error(w, r, http.StatusInternalServerError, response.CodeInternal, "внутренняя ошибка сервиса")
	}
}
