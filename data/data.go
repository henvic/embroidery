package data

type Address struct {
	AddressID    string
	ClientID     string
	Name         string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Country      string
	ZipCode      string
	Phone        string
	Status       string
}

type Asset struct {
	AssetID          string
	ClientID         string
	Filepath         string
	Status           string
	OriginalFilePath string
	ReceivedDate     string
}

type Authentication struct {
	EmployeeID  string
	Email       string
	Provider    string
	AccessLevel int
}

type Client struct {
	ClientID  string
	FirstName string
	LastName  string
	Email     string
	Status    string
}

type GoodsRecord struct {
	GoodsID    string
	JobID      string
	EmployeeID string
	OwnerID    string
	Type       string
	Amount     int
	Unit       string
	Notes      string
	Date       string
	Status     int
}

type Job struct {
	JobID      string
	OrderID    string
	ClientID   string
	AssetID    string
	Status     string
	Type       string
	Amount     int
	Price      int
	StartTime  string
	EndTime    string
	Complexity int
}
