package query

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xuroi/xuroi/api/internal/models"
)

func (r *Reader) PostAdminAudit(ctx context.Context, postID string) (models.PostAdminAudit, error) {
	var audit models.PostAdminAudit
	err := r.pool.QueryRow(ctx, `
		SELECT p.id, p.thread_id, t.title, p.author_id, a.display_name, p.author_ip, p.created_at, p.edited_at,
		       p.reaction_count, p.is_op,
		       (SELECT COUNT(*)::int FROM post_revisions pr WHERE pr.post_id = p.id)
		FROM posts p
		JOIN threads t ON t.id = p.thread_id
		JOIN actors a ON a.id = p.author_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, postID).Scan(
		&audit.PostID, &audit.ThreadID, &audit.ThreadTitle, &audit.AuthorID, &audit.AuthorName, &audit.AuthorIP,
		&audit.CreatedAt, &audit.EditedAt, &audit.ReactionCount, &audit.IsOP, &audit.RevisionCount,
	)
	if err == pgx.ErrNoRows {
		return models.PostAdminAudit{}, ErrNotFound
	}
	return audit, err
}

func (r *Reader) PostRevisions(ctx context.Context, postID string) ([]models.PostRevision, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pr.revision, pr.body_html, pr.edited_at, COALESCE(a.display_name, 'Unknown')
		FROM post_revisions pr
		LEFT JOIN actors a ON a.id = pr.editor_id
		WHERE pr.post_id = $1
		ORDER BY pr.revision DESC
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []models.PostRevision
	for rows.Next() {
		var rev models.PostRevision
		if err := rows.Scan(&rev.Revision, &rev.BodyHTML, &rev.EditedAt, &rev.EditorName); err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	return revisions, rows.Err()
}