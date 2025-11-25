package models

type Album struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	CoverURL string `json:"coverUrl"`
	URL      string `json:"url"`
	IsFree   bool   `json:"isFree"`
	IsNYP    bool   `json:"isNyp"` // Name Your Price
	Price    string `json:"price"`
	Status   string `json:"status"` // "free", "nyp", "paid"
}
