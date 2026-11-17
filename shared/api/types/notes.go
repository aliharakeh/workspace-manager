package types

// NoteItem preserves checklist completion for the GUI and agent clients.
type NoteItem struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type Note struct {
	ID        int64      `json:"id"`
	AppID     int64      `json:"app_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Items     []NoteItem `json:"items"`
	ImagePath string     `json:"image_path"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

// NoteSaveInput replaces the title, content and checklist. Omitted image_data
// retains the current image; an empty string removes it; base64 replaces it.
type NoteSaveInput struct {
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Items     []NoteItem `json:"items"`
	ImageData *string    `json:"image_data,omitempty"`
}
