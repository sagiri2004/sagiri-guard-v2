package firewall

import "sync"

var (
	categoryMu      sync.RWMutex
	CategoryDomains = make(map[string][]string)
)

// SetCategoryDomains cập nhật danh sách domain theo category từ backend
func SetCategoryDomains(m map[string][]string) {
	categoryMu.Lock()
	defer categoryMu.Unlock()
	CategoryDomains = m
}

// GetDomainsByCategory lấy danh sách domain theo category
func GetDomainsByCategory(category string) []string {
	categoryMu.RLock()
	defer categoryMu.RUnlock()
	return CategoryDomains[category]
}

// GetAllCategories lấy tất cả category names
func GetAllCategories() []string {
	categoryMu.RLock()
	defer categoryMu.RUnlock()
	cats := make([]string, 0, len(CategoryDomains))
	for cat := range CategoryDomains {
		cats = append(cats, cat)
	}
	return cats
}
