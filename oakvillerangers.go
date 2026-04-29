package main

type OakvilleRangersScraper struct{ HTTPFetcher }

func (s *OakvilleRangersScraper) Name() string     { return "Oakville Rangers" }
func (s *OakvilleRangersScraper) SiteURL() string  { return "https://oakvillerangers.ca" }
func (s *OakvilleRangersScraper) PageURLs() []string { return []string{s.SiteURL() + "/Articles/"} }
