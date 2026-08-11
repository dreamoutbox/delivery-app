package models

import (
	"time"
)

// User Roles
const (
	RoleAdmin      = "admin"
	RoleStaff      = "staff"
	RoleRestaurant = "restaurant"
	RoleRider      = "rider"
	RoleCustomer   = "customer"
)

// Order Statuses
const (
	StatusPending        = "pending"
	StatusConfirmed      = "confirmed"
	StatusReadyForPickup = "ready_for_pickup"
	StatusInDelivery     = "in_delivery"
	StatusDelivered      = "delivered"
	StatusCancelled      = "cancelled"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(50);not null;default:'customer'" json:"role"`
	Phone        string    `gorm:"type:varchar(50)" json:"phone"`
	Address      string    `gorm:"type:text" json:"address"`
	Lat          *float64  `gorm:"type:decimal(10,8)" json:"lat"`
	Lng          *float64  `gorm:"type:decimal(11,8)" json:"lng"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Restaurant struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	OwnerID               uint       `gorm:"not null;index" json:"owner_id"`
	Owner                 User       `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Name                  string     `gorm:"type:varchar(255);not null" json:"name"`
	Description           string     `gorm:"type:text" json:"description"`
	Address               string     `gorm:"type:text;not null" json:"address"`
	Lat                   float64    `gorm:"type:decimal(10,8);not null" json:"lat"`
	Lng                   float64    `gorm:"type:decimal(11,8);not null" json:"lng"`
	ImagePath             string     `gorm:"type:varchar(500)" json:"image_path"`
	DeliveryBaseFeeSatang int        `gorm:"not null;default:0" json:"delivery_base_fee_satang"`
	DeliveryPerKmSatang   int        `gorm:"not null;default:0" json:"delivery_per_km_satang"`
	IsOpen                bool       `gorm:"not null;default:true" json:"is_open"`
	MenuItems             []MenuItem `gorm:"foreignKey:RestaurantID" json:"menu_items,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type MenuItem struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RestaurantID uint      `gorm:"not null;index" json:"restaurant_id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	PriceSatang  int       `gorm:"not null" json:"price_satang"`
	ImagePath    string    `gorm:"type:varchar(500)" json:"image_path"`
	IsAvailable  bool      `gorm:"not null;default:true" json:"is_available"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Order struct {
	ID                 uint        `gorm:"primaryKey" json:"id"`
	CustomerID         uint        `gorm:"not null;index" json:"customer_id"`
	Customer           User        `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	RestaurantID       uint        `gorm:"not null;index" json:"restaurant_id"`
	Restaurant         Restaurant  `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	RiderID            *uint       `gorm:"index" json:"rider_id"`
	Rider              *User       `gorm:"foreignKey:RiderID" json:"rider,omitempty"`
	Status             string      `gorm:"type:order_status;not null;default:'pending'" json:"status"`
	DeliveryAddress    string      `gorm:"type:text;not null" json:"delivery_address"`
	DeliveryLat        float64     `gorm:"type:decimal(10,8);not null" json:"delivery_lat"`
	DeliveryLng        float64     `gorm:"type:decimal(11,8);not null" json:"delivery_lng"`
	SubtotalSatang     int         `gorm:"not null" json:"subtotal_satang"`
	DeliveryFeeSatang  int         `gorm:"not null" json:"delivery_fee_satang"`
	TotalSatang        int         `gorm:"not null" json:"total_satang"`
	RiderPickupProof   string      `gorm:"type:varchar(500)" json:"rider_pickup_proof"`
	RiderDeliveryProof string      `gorm:"type:varchar(500)" json:"rider_delivery_proof"`
	Note               string      `gorm:"type:text" json:"note"`
	OrderItems         []OrderItem `gorm:"foreignKey:OrderID" json:"order_items,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	OrderID     uint   `gorm:"not null;index" json:"order_id"`
	MenuItemID  uint   `gorm:"not null" json:"menu_item_id"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	PriceSatang int    `gorm:"not null" json:"price_satang"`
	Quantity    int    `gorm:"not null" json:"quantity"`
}
