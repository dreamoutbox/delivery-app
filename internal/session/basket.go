package session

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type BasketData struct {
	RestaurantID uint         `json:"restaurant_id"`
	Items        map[uint]int `json:"items"` // menuItemId -> qty
}

func GetBasket(c *gin.Context) (uint, map[uint]int) {
	sess, err := Store.Get(c.Request, SessionName)
	if err != nil {
		return 0, make(map[uint]int)
	}

	val, ok := sess.Values["basket_json"].(string)
	if !ok || val == "" {
		return 0, make(map[uint]int)
	}

	var data BasketData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return 0, make(map[uint]int)
	}

	if data.Items == nil {
		data.Items = make(map[uint]int)
	}

	return data.RestaurantID, data.Items
}

func SaveBasket(c *gin.Context, restaurantID uint, items map[uint]int) error {
	sess, err := Store.Get(c.Request, SessionName)
	if err != nil {
		sess, _ = Store.New(c.Request, SessionName)
	}

	if items == nil {
		items = make(map[uint]int)
	}

	data := BasketData{
		RestaurantID: restaurantID,
		Items:        items,
	}

	bytesVal, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sess.Values["basket_json"] = string(bytesVal)
	return Store.Save(c.Request, c.Writer, sess)
}

func ClearBasket(c *gin.Context) error {
	sess, err := Store.Get(c.Request, SessionName)
	if err != nil {
		return nil
	}

	delete(sess.Values, "basket_json")
	return Store.Save(c.Request, c.Writer, sess)
}

func GetBasketCount(c *gin.Context) int {
	_, items := GetBasket(c)
	total := 0
	for _, qty := range items {
		total += qty
	}
	return total
}
