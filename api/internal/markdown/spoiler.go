package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

var spoilerRe = regexp.MustCompile(`>!([^!\n]+)!<`)

func expandSpoilers(source string) string {
	return spoilerRe.ReplaceAllStringFunc(source, func(m string) string {
		sub := spoilerRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		inner := strings.TrimSpace(sub[1])
		if inner == "" {
			return m
		}
		return fmt.Sprintf(
			"\n\n<details class=\"spoiler\"><summary>Spoiler</summary>\n\n%s\n\n</details>\n\n",
			inner,
		)
	})
}