package api

import (
	"strings"
	"testing"
)

func TestBuildBackfillQualityQueryPartsUseSingularTableNames(t *testing.T) {
	t.Parallel()

	parts := buildBackfillQualityQueryParts()

	if parts.pageTable != "page" {
		t.Fatalf("expected page table name %q, got %q", "page", parts.pageTable)
	}
	if parts.videoTable != "video" {
		t.Fatalf("expected video table name %q, got %q", "video", parts.videoTable)
	}
	if strings.Contains(parts.selectClause, "videos.") {
		t.Fatalf("expected select clause to avoid plural video table names, got %q", parts.selectClause)
	}
	if strings.Contains(parts.joinClause, "JOIN videos") {
		t.Fatalf("expected join clause to avoid plural video table names, got %q", parts.joinClause)
	}
	if !strings.Contains(parts.selectClause, "video.name as video_name") {
		t.Fatalf("expected select clause to reference video table, got %q", parts.selectClause)
	}
	if !strings.Contains(parts.joinClause, "JOIN video ON video.id = page.video_id") {
		t.Fatalf("expected join clause to use singular table names, got %q", parts.joinClause)
	}
	if parts.whereClause != "page.download_status = ? AND (page.quality = 0 OR page.width = 0)" {
		t.Fatalf("unexpected where clause: %q", parts.whereClause)
	}
}
