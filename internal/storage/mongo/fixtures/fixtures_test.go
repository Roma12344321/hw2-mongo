package fixtures_test

import (
	"testing"

	"hw2-mongo/internal/storage/mongo/fixtures"
)

func TestProductsCountAndUniqueness(t *testing.T) {
	products := fixtures.Products()
	if len(products) != 6 {
		t.Fatalf("ожидалось 6 товаров, получено %d", len(products))
	}

	sizes := map[string]int{}
	for _, product := range products {
		if len(product.Variants) != 2 {
			t.Errorf("у %s ожидались варианты L и M", product.ID)
		}
		if product.Variants[0].Size != "L" || product.Variants[1].Size != "M" {
			t.Errorf("у %s порядок вариантов должен быть L, M", product.ID)
		}
		sizes[product.ID]++
	}
	if len(sizes) != 6 {
		t.Error("артикулы товаров должны быть уникальны")
	}
}

func TestOrdersCount(t *testing.T) {
	if orders := fixtures.Orders(); len(orders) != 35 {
		t.Fatalf("ожидалось 35 заказов, получено %d", len(orders))
	}
}

func TestLargeOrdersCount(t *testing.T) {
	if orders := fixtures.LargeOrders(); len(orders) != 10000 {
		t.Fatalf("ожидалось 10000 заказов, получено %d", len(orders))
	}
}

func TestHistoryExpectedOrder(t *testing.T) {
	expected := fixtures.HistoryExpected()
	if len(expected) != 20 {
		t.Fatalf("ожидалось 20 id, получено %d", len(expected))
	}
	if expected[0] != "H24" || expected[1] != "H25" || expected[18] != "H06" || expected[19] != "H07" {
		t.Errorf("неожиданный порядок истории: %v", expected)
	}
}

func TestSearchCaseCounterexample(t *testing.T) {
	var clothesL2 []string
	for _, testCase := range fixtures.SearchCases() {
		if testCase.Category == "clothes" && testCase.Size == "L" && testCase.Qty == 2 {
			clothesL2 = testCase.ExpectedID
		}
	}
	if len(clothesL2) != 2 || clothesL2[0] != "B" || clothesL2[1] != "C" {
		t.Errorf("поиск clothes/L/2 должен вернуть [B C], получено %v", clothesL2)
	}

	products := fixtures.Products()
	for _, product := range products {
		if product.ID == "A" {
			for _, variant := range product.Variants {
				if variant.Size == "L" && variant.Stock != 0 {
					t.Error("у A остаток L должен быть 0 в исходном датасете")
				}
			}
		}
	}
}

func TestReportExpectedTopOrder(t *testing.T) {
	expected := fixtures.ReportExpected()
	if len(expected) != 5 {
		t.Fatalf("ожидалось 5 строк, получено %d", len(expected))
	}
	if expected[0].SKU != "A" || expected[1].SKU != "B" {
		t.Error("при равной выручке сортировка по sku: A перед B")
	}
	if expected[0].Revenue != "320.00" {
		t.Errorf("выручка A должна быть 320.00, получено %s", expected[0].Revenue)
	}
}

func TestDecimalPrecision(t *testing.T) {
	if got := fixtures.Decimal("320.00").String(); got != "320.00" {
		t.Errorf("Decimal128 должен сохранять масштаб: получено %s", got)
	}
}
