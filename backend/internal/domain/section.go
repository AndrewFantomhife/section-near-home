package domain

// Теги JSON точно соответствуют полям объектов из mock-api/db.json, иначе декодирование молча обнулит поля.

import "fmt"

type Section struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Sport    string `json:"sport"`
	Name     string `json:"name"`
	District string `json:"district"`
	Address  string `json:"address"`
	Coach    string `json:"coach"`
	Schedule string `json:"schedule"`
	MinAge   int    `json:"minAge"`
	Price    int    `json:"price"`
	Spots    int    `json:"spots"`
}

func (section Section) HasFreeSpots() bool {
	return section.Spots > 0
}

func (section Section) IsFree() bool {
	return section.Price == 0
}

func (section Section) FitsAge(childAge int) bool {
	return childAge >= section.MinAge
}

func (section Section) FormatPrice() string {
	if section.IsFree() {
		return "бесплатно"
	}
	return fmt.Sprintf("%d руб/мес", section.Price)
}
