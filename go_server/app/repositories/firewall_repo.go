package repositories

import (
	"demo/network/go_server/app/models"

	"gorm.io/gorm"
)

type FirewallRepository struct {
	DB *gorm.DB
}

func NewFirewallRepository(db *gorm.DB) *FirewallRepository {
	return &FirewallRepository{DB: db}
}

func (r *FirewallRepository) GetDomainsByCategories(catIDs []int) ([]string, error) {
	var domains []string
	if len(catIDs) == 0 {
		return domains, nil
	}
	result := r.DB.Model(&models.FirewallDomain{}).
		Where("category_id IN ?", catIDs).
		Pluck("domain", &domains)
	return domains, result.Error
}
