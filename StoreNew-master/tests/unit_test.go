package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

type Cigarette struct {
	Brand    string  `json:"brand"`
	Type     string  `json:"type"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	PhotoURL string  `json:"photo_url"`
}

// 7.1 Unit тест: Проверка добавления товара в корзину
func TestAddCigaretteToCart_Unit(t *testing.T) {
	setupTestDB() // Важно вызвать настройку базы данных перед тестами
	clearCollections()

	cigarette := Cigarette{
		Brand:    "BrandTest",
		Type:     "TypeTest",
		Price:    10.99,
		Category: "CategoryTest",
		PhotoURL: "http://example.com/photo.jpg",
	}

	_, err := cartCollection.InsertOne(context.Background(), cigarette)
	assert.NoError(t, err)

	var result Cigarette
	err = cartCollection.FindOne(context.Background(), bson.M{"brand": "BrandTest"}).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "BrandTest", result.Brand)
}
