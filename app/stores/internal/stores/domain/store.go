// Package domain holds the store type. It imports nothing but the standard
// library: no database, no HTTP.
package domain

// Store is a shop with the details the dashboard shows about it. The last two
// fields are computed, the rest are Pagila's own columns.
type Store struct {
	ID    int
	Label string // "City, Country"

	Manager    string
	Phone      string
	Address    string
	District   string
	PostalCode string
	City       string
	Country    string

	InventoryItems  int
	ActiveCustomers int
}
