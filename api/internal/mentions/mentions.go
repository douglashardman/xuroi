package mentions

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/models"
	"github.com/xuroi/xuroi/api/internal/slug"
)

var (
	quotedRe  = regexp.MustCompile(`@"([^"]+)"`)
	bracketRe = regexp.MustCompile(`@\[([^\]]+)\]`)
	slugRe    = regexp.MustCompile(`@([a-zA-Z0-9][a-zA-Z0-9_-]*)`)
)

type Actor struct {
	ID          string
	DisplayName string
}

type Index struct {
	bySlug map[string]Actor
	byName map[string]Actor
}

type Result struct {
	Markdown string
	ActorIDs []string
}

// LoadIndex builds a lookup of members and agents by slug and display name.
func LoadIndex(ctx context.Context, pool *pgxpool.Pool) (*Index, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, display_name
		FROM actors
		WHERE type IN ('human', 'agent') AND deleted_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("list actors: %w", err)
	}
	defer rows.Close()

	idx := &Index{
		bySlug: make(map[string]Actor),
		byName: make(map[string]Actor),
	}
	for rows.Next() {
		var a Actor
		if err := rows.Scan(&a.ID, &a.DisplayName); err != nil {
			return nil, fmt.Errorf("scan actor: %w", err)
		}
		s := strings.ToLower(slug.FromDisplayName(a.DisplayName))
		if s != "" {
			idx.bySlug[s] = a
		}
		idx.byName[strings.ToLower(strings.TrimSpace(a.DisplayName))] = a
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return idx, nil
}

func (idx *Index) resolve(token string) (Actor, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Actor{}, false
	}
	if a, ok := idx.byName[strings.ToLower(token)]; ok {
		return a, true
	}
	if a, ok := idx.bySlug[strings.ToLower(token)]; ok {
		return a, true
	}
	return Actor{}, false
}

// Expand rewrites @mentions to profile links and returns mentioned actor IDs.
func Expand(source string, idx *Index) Result {
	if source == "" || idx == nil {
		return Result{Markdown: source}
	}

	seen := make(map[string]struct{})
	var ids []string
	mark := func(a Actor) string {
		if _, ok := seen[a.ID]; !ok {
			seen[a.ID] = struct{}{}
			ids = append(ids, a.ID)
		}
		return fmt.Sprintf("[@%s](%s)", a.DisplayName, models.UserURL(a.DisplayName))
	}

	out := source
	out = quotedRe.ReplaceAllStringFunc(out, func(m string) string {
		sub := quotedRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		if a, ok := idx.resolve(sub[1]); ok {
			return mark(a)
		}
		return m
	})
	out = bracketRe.ReplaceAllStringFunc(out, func(m string) string {
		sub := bracketRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		if a, ok := idx.resolve(sub[1]); ok {
			return mark(a)
		}
		return m
	})
	out = replaceSlugMentions(out, idx, mark)
	return Result{Markdown: out, ActorIDs: ids}
}

func replaceSlugMentions(source string, idx *Index, mark func(Actor) string) string {
	var b strings.Builder
	rest := source
	for {
		loc := slugRe.FindStringIndex(rest)
		if loc == nil {
			b.WriteString(rest)
			break
		}
		start, end := loc[0], loc[1]
		if start > 0 {
			prev := rest[start-1]
			if prev == '[' || !mentionPrefixOK(rune(prev)) {
				b.WriteString(rest[:end])
				rest = rest[end:]
				continue
			}
		}
		b.WriteString(rest[:start])
		token := rest[start+1 : end]
		if a, ok := idx.resolve(token); ok {
			b.WriteString(mark(a))
		} else {
			b.WriteString(rest[start:end])
		}
		rest = rest[end:]
	}
	return b.String()
}

func mentionPrefixOK(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("([{>.,!?:;-\n\r\t", r)
}

// FilterSelf removes the author's own ID from a mention list.
func FilterSelf(actorIDs []string, authorID string) []string {
	if authorID == "" || len(actorIDs) == 0 {
		return actorIDs
	}
	out := make([]string, 0, len(actorIDs))
	for _, id := range actorIDs {
		if id != authorID {
			out = append(out, id)
		}
	}
	return out
}