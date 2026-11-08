package provider

import "testing"

func TestStreamLinks(t *testing.T) {
	for _, tt := range []struct {
		name, data, stream, subtitle string
	}{
		{"actual embed shape", `{"src":"https://video.example/master.m3u8","subtitles":[{"lang":"en","label":"English","default":true,"src":"https://subs.example/en.vtt"}]}`, "https://video.example/master.m3u8", "https://subs.example/en.vtt"},
		{"default subtitle", `{"sources":[{"src":"https:\/\/video.example\/master.m3u8"}],"subtitles":[{"src":"https:\/\/subs.example\/fr.vtt","default":false},{"src":"https:\/\/subs.example\/en.vtt","default":true}]}`, "https://video.example/master.m3u8", "https://subs.example/en.vtt"},
		{"default beats english", `{"sources":[{"src":"https://video.example/master.m3u8"}],"subtitles":[{"src":"https://subs.example/en.vtt","lang":"English"},{"src":"https://subs.example/fr.vtt","default":true}]}`, "https://video.example/master.m3u8", "https://subs.example/fr.vtt"},
		{"english fallback", `{"sources":[{"src":"https://video.example/master.m3u8"}],"subtitles":[{"src":"https://subs.example/fr.vtt","lang":"French"},{"src":"https://subs.example/en.vtt","lang":"English"}]}`, "https://video.example/master.m3u8", "https://subs.example/en.vtt"},
		{"no subtitle", `{"sources":[{"src":"https://video.example/master.m3u8"}]}`, "https://video.example/master.m3u8", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stream, subtitle, err := streamLinks(tt.data)
			if err != nil || stream != tt.stream || subtitle != tt.subtitle {
				t.Fatalf("streamLinks() = %q, %q, %v; want %q, %q", stream, subtitle, err, tt.stream, tt.subtitle)
			}
		})
	}
}
