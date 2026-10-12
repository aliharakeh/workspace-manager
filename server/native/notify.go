package native

import (
	"context"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Notify sends a best-effort OS notification. It never returns an error:
// notification failures must not interrupt the runner.
func Notify(ctx context.Context, title, body string) {
	title = strings.TrimSpace(title)
	if ctx == nil || title == "" {
		return
	}
	if !runtime.IsNotificationAvailable(ctx) {
		return
	}
	_ = runtime.SendNotification(ctx, runtime.NotificationOptions{
		Title: title,
		Body:  body,
	})
}
