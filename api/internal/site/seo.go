package site

type SEOPolicy struct {
	NofollowUserLinks bool `json:"nofollow_user_links"`
}

func DefaultSEOPolicy() SEOPolicy {
	return SEOPolicy{NofollowUserLinks: true}
}