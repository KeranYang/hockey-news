package main

type CanlanOakvilleScraper struct{ HTTPFetcher }

func (s *CanlanOakvilleScraper) Name() string     { return "Canlan Sports Oakville" }
func (s *CanlanOakvilleScraper) SiteURL() string  { return "https://www.canlansports.com" }
func (s *CanlanOakvilleScraper) PageURLs() []string { return []string{"https://www.canlansports.com/news/"} }
