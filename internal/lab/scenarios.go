package lab

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"hw2-mongo/internal/storage/mongo/fixtures"
)

var scenarios = map[string]ScenarioFunc{
	"data":     DataScenario,
	"index":    IndexScenario,
	"purchase": PurchaseScenario,
	"failover": FailoverScenario,
	"reset":    ResetScenario,
}

func PrintFixtures() {
	Log("PRODUCTS", fixtures.Products())
	Log("ORDERS", fixtures.Orders())

	searchExpected := make(bson.A, 0)
	for _, testCase := range fixtures.SearchCases() {
		searchExpected = append(searchExpected, bson.D{
			{Key: "input", Value: bson.D{
				{Key: "category", Value: testCase.Category},
				{Key: "size", Value: testCase.Size},
				{Key: "qty", Value: testCase.Qty},
			}},
			{Key: "ids", Value: testCase.ExpectedID},
		})
	}

	Log("SEARCH EXPECTED", searchExpected)
	Log("HISTORY EXPECTED", bson.D{
		{Key: "input", Value: bson.D{
			{Key: "userId", Value: int32(2500)},
			{Key: "status", Value: "paid"},
		}},
		{Key: "ids", Value: fixtures.HistoryExpected()},
	})

	reportRows := make(bson.A, 0, 5)
	for _, row := range fixtures.ReportExpected() {
		reportRows = append(reportRows, bson.D{
			{Key: "sku", Value: row.SKU},
			{Key: "name", Value: row.Name},
			{Key: "qty", Value: row.Qty},
			{Key: "revenue", Value: row.Revenue},
		})
	}

	Log("REPORT EXPECTED", bson.D{
		{Key: "from", Value: fixtures.ReportFrom},
		{Key: "to", Value: fixtures.ReportTo},
		{Key: "rows", Value: reportRows},
	})
}

func PrintHelp() {
	fmt.Print(`MongoDB — домашняя работа (Go, официальный драйвер)

  make up                    Поднять три узла и инициализировать replica set
  make reset                 Восстановить 6 товаров и 35 заказов; сбросить результаты
  make reset-index           Загрузить 10 000 заказов для опыта с индексом
  make fixtures              Показать контрольные данные и эталоны
  make data                  Проверить валидатор, поиск, историю и отчёт
  make index                 Сравнить один запрос до/после индекса на 10 000 заказов
  make purchase              Проверить покупку, откат, повтор и конфликт
  make failover              Сменить primary, прочитать и записать тем же клиентом
  make all                   Последовательно выполнить четыре проверки
  make down                  Остановить стенд, сохранив данные
  make clean                 Удалить контейнеры и данные стенда

Эквивалент любой make-цели: docker compose run --rm app <команда>.
Каждый сценарий сам готовит свои данные; запускайте последовательно.
Все результаты и полные explain: results/results.json (Extended JSON).
Базы стенда: hw2, hw2_failover. Пересборка после правок: make docker-build.

URI внутри Docker-сети стенда:
mongodb://m1:27017,m2:27017,m3:27017/hw2?replicaSet=hw2rs&retryWrites=true&serverSelectionTimeoutMS=10000&readPreference=primary
`)
}

func RunScenarioNames(command string) ([]string, bool) {
	switch command {
	case "all":
		return ScenarioNames, true
	case "data", "index", "purchase", "failover", "reset":
		return []string{command}, true
	default:
		return nil, false
	}
}
