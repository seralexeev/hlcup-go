package main

type User struct {
	ID, BirthDate                      int
	Age                                int `json:"-,"`
	Email, FirstName, LastName, Gender string

	visits map[int]*Visit `json:"-,"`
}

func (user *User) IsValid() bool {
	return user.ID > 0 && user.Email != "" && user.FirstName != "" && user.LastName != "" && user.Gender != ""
}

func (user *User) CalculateAge() {
	user.Age = (currentDate - user.BirthDate) / 31557600
}

func readUser(data []byte) (*User, error) {
	var user User
	return &user, user.UnmarshalJSON(data)
}

type Location struct {
	ID, Distance         int
	Place, Country, City string

	visits map[int]*Visit `json:"-,"`
}

func (location *Location) IsValid() bool {
	return location.ID > 0 && location.Place != "" && location.Country != "" && location.City != ""
}

func readLocation(data []byte) (*Location, error) {
	var location Location
	return &location, location.UnmarshalJSON(data)
}

type Visit struct {
	ID, Location, User, VisitedAt, Mark int

	userRef     *User     `json:"-,"`
	locationRef *Location `json:"-,"`
}

func (visit *Visit) IsValid() bool {
	return visit.ID > 0
}

func readVisit(data []byte) (*Visit, error) {
	var visit Visit
	return &visit, visit.UnmarshalJSON(data)
}

type VisitsResult struct {
	Visits []VisitResult
}

type VisitResult struct {
	Mark, VisitedAt int
	Place           string
}

type AvgResult struct {
	Avg float64
}

type UsersFile struct {
	Users []*User
}

type LocationsFile struct {
	Locations []*Location
}

type VisitsFile struct {
	Visits []*Visit
}
