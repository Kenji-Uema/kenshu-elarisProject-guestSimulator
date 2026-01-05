package domain

type Cottage struct {
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Details       CottageDetails `json:"details"`
	Photos        []string       `json:"photos"`
	PricePerNight float32        `json:"price_per_night"`
}

type CottageDetails struct {
	Description          string `json:"description"`
	View                 string `json:"view"`
	FurnitureDescription string `json:"furniture_description"`
	BathroomDescription  string `json:"bathroom_description"`
	AmenitiesDescription string `json:"amenities_description"`
}

type CottageAvailable struct {
	CottageName string
	Periods     []Period
}
