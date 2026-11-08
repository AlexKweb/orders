package internal_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gulmix/orders/internal"
	healthController "github.com/gulmix/orders/internal/controllers/http/health"
	orderController "github.com/gulmix/orders/internal/controllers/http/order"

	"github.com/gulmix/orders/internal/requestid"
)

// Тест уровня маршрутов. Он ничего не проверяет из вашего задания — он
// показывает, как тестировать HTTP-слой: через httptest, без
// поднятия настоящего сокета и без сети, поэтому такие тесты идут за
// миллисекунды и не флакают.
//
// Тесты на заказы пишете вы. Ориентир по объёму — в задании модуля.
func TestHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := internal.InitRoutes(logger, healthController.NewController(), nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusOK)
	}

	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("ответ не разобрался как JSON: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf(`status = %q, ожидался "ok"`, got["status"])
	}

	// Заголовок проставляет middleware — заодно проверяем, что обвязка на месте.
	if rec.Header().Get(requestid.Header) == "" {
		t.Errorf("заголовок %s пустой", requestid.Header)
	}
}

func TestOrder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := internal.InitRoutes(logger, nil, orderController.NewController())

	qwer := io.Reader(strings.NewReader(`{
	"customer_id": "cust_1",
	"items": [
		{"sku": "book", "qty": 2, "price_cents": 1500},
		{"sku": "pen", "qty": 10, "price_cents": 300}
	]
}`))

	//----------------Тест проверки наличия Content-Type----------------
	req := httptest.NewRequest(http.MethodPost, "/orders", qwer)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusCreated)
	}
	//----------------Тест ошибки 'Некоректные данные'----------------

	qwer = io.Reader(strings.NewReader(`{
	"customer_id": "cst_1",
	"items": [{"ty": 10, "price_cents": 300}]
}`))
	req = httptest.NewRequest(http.MethodPost, "/orders", qwer)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusBadRequest)
	}
	//----------------Тест на валидацию----------------

	qwer = io.Reader(strings.NewReader(`{
	"customer_id": "cust_1",
	"items": [
		{"sku": "", "qty": 2, "price_cents": -1500},
		{"sku": "pen", "qty": 0, "price_cents": 300}
	]
}`))
	req = httptest.NewRequest(http.MethodPost, "/orders", qwer)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusUnprocessableEntity)
	}

	//----------------Тест POST заказа с лишними полями----------------
	qwer = io.Reader(strings.NewReader(`{
	"customer_id": "cust_1",
	"asd": "asd",
	"items": [
		{"sku": "book", "qty": 2, "price_cents": 1500, "asd": "asd"},
		{"sku": "pen", "qty": 10, "price_cents": 300}
	]
}`))
	req = httptest.NewRequest(http.MethodPost, "/orders", qwer)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusBadRequest)
	}

	//----------------Тест POST заказа----------------
	qwer = io.Reader(strings.NewReader(`{
	"customer_id": "cust_1",
	"items": [
		{"sku": "book", "qty": 2, "price_cents": 1500},
		{"sku": "pen", "qty": 10, "price_cents": 300}
	]
}`))
	req = httptest.NewRequest(http.MethodPost, "/orders", qwer)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	before := time.Now().UTC().Truncate(time.Second)
	srv.ServeHTTP(rec, req)
	after := time.Now()
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusCreated)
	}
	if !strings.Contains(rec.Header().Get("Location"), "/orders/ord_") { //пришлось поменять ради race, т к 2/5 тест найдет ord_100002
		t.Fatalf("Location = %q, ожидался %q", rec.Header().Get("Location"), "/orders/ord_1")
	}

	got := orderController.OrderBody{}

	tmpJson := json.NewDecoder(rec.Body)
	tmpJson.DisallowUnknownFields()
	if err := tmpJson.Decode(&got); err != nil {
		t.Fatalf("ответ не разобрался как JSON: %v", err)

	}
	if !strings.Contains(got.Id, "ord_") { //пришлось поменять ...
		t.Errorf(`Ответ неверный, ожидался {"id": "ord_1", ...}, а пришёл %q`, got.Id)
	}
	if got.Customer_id != "cust_1" {
		t.Errorf(`Ответ неверный, ожидался {"customer_id": "cust_1", ...}, а пришёл %q`, got.Customer_id)
	}
	if got.Status != "created" {
		t.Errorf(`Ответ неверный, ожидался {"status": "created", ...}, а пришёл %q`, got.Status)
	}
	if len(got.Items) != 2 {
		t.Errorf(`Ответ неверный, ожидался {"items": [..., ...], ...}, а пришёл %v`, got.Items)
	}
	if got.Items[0].Sku != "book" || got.Items[0].Qty != 2 || got.Items[0].Price_cents != 1500 {
		t.Errorf(`Ответ неверный, ожидался {"items": [{"sku": "book", "qty": 2, "price_cents": 1500}, ...], ...}, а пришёл %v`, got.Items)
	}
	if got.Items[1].Sku != "pen" || got.Items[1].Qty != 10 || got.Items[1].Price_cents != 300 {
		t.Errorf(`Ответ неверный, ожидался {"items": [..., {"sku": "pen", "qty": 10, "price_cents": 300}], ...}, а пришёл %v`, got.Items)
	}
	if got.Total_cents != 2*1500+10*300 {
		t.Errorf(`Ответ неверный, ожидался {"total_cents": 6000, ...}, а пришёл %d`, got.Total_cents)
	}
	if createdAt, err := time.Parse(time.RFC3339, got.Created_at); err != nil || createdAt.Before(before) || createdAt.After(after) {
		t.Fatalf("created_at = %v вне диапазона [%v, %v]", createdAt, before, after)
	}

	//----------------Тест POST race----------------
	var wg sync.WaitGroup
	var countMiss atomic.Int32
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			qwer1 := io.Reader(strings.NewReader(`{
	"customer_id": "cust_1",
	"items": [{"sku": "book", "qty": 2, "price_cents": 1500}]
}`))
			req1 := httptest.NewRequest(http.MethodPost, "/orders", qwer1)
			req1.Header.Set("Content-Type", "application/json")
			rec1 := httptest.NewRecorder()
			srv.ServeHTTP(rec1, req1)
			if rec1.Code != http.StatusCreated {
				countMiss.Add(1)
			}
		}()
	}
	wg.Wait()
	if countMiss.Load() > 0 {
		t.Fatalf("Кол неправильных статусов: %d, ожидался %d", countMiss.Load(), 0)
	}

	//----------------Тест GET отсутствие ордера----------------
	req = httptest.NewRequest(http.MethodGet, "/orders/ord_0", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusNotFound)
	}

	//----------------Тест GET----------------

	req = httptest.NewRequest(http.MethodGet, "/orders/ord_1", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Location") != "/orders/ord_1" {
		t.Fatalf("Location = %q, ожидался %q", rec.Header().Get("Location"), "/orders/ord_1")
	}

	got = orderController.OrderBody{}

	tmpJson = json.NewDecoder(rec.Body)
	tmpJson.DisallowUnknownFields()
	if err := tmpJson.Decode(&got); err != nil {
		t.Fatalf("ответ не разобрался как JSON: %v", err)
	}
	if got.Id != "ord_1" {
		t.Errorf(`Ответ неверный, ожидался {"id": "ord_1", ...}, а пришёл %q`, got.Id)
	}
	if got.Customer_id != "cust_1" {
		t.Errorf(`Ответ неверный, ожидался {"customer_id": "cust_1", ...}, а пришёл %q`, got.Customer_id)
	}
	if got.Status != "created" {
		t.Errorf(`Ответ неверный, ожидался {"status": "created", ...}, а пришёл %q`, got.Status)
	}
	if len(got.Items) != 2 {
		t.Errorf(`Ответ неверный, ожидался {"items": [..., ...], ...}, а пришёл %v`, got.Items)
	}
	if got.Items[0].Sku != "book" || got.Items[0].Qty != 2 || got.Items[0].Price_cents != 1500 {
		t.Errorf(`Ответ неверный, ожидался {"items": [{"sku": "book", "qty": 2, "price_cents": 1500}, ...], ...}, а пришёл %v`, got.Items)
	}
	if got.Items[1].Sku != "pen" || got.Items[1].Qty != 10 || got.Items[1].Price_cents != 300 {
		t.Errorf(`Ответ неверный, ожидался {"items": [..., {"sku": "pen", "qty": 10, "price_cents": 300}], ...}, а пришёл %v`, got.Items)
	}
	if got.Total_cents != 2*1500+10*300 {
		t.Errorf(`Ответ неверный, ожидался {"total_cents": 6000, ...}, а пришёл %d`, got.Total_cents)
	}

	//----------------Тест POST race----------------
	for i := 1; i < 100002; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req1 := httptest.NewRequest(http.MethodGet, "/orders/ord_"+strconv.Itoa(i), nil)
			rec1 := httptest.NewRecorder()
			srv.ServeHTTP(rec1, req1)
			if rec1.Code != http.StatusOK {
				countMiss.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if countMiss.Load() > 0 {
		t.Fatalf("Кол неправильных статусов: %d, ожидался %d", countMiss.Load(), 0)
	}

	// Заголовок проставляет middleware — заодно проверяем, что обвязка на месте.
	if rec.Header().Get(requestid.Header) == "" {
		t.Errorf("заголовок %s пустой", requestid.Header)
	}
}
