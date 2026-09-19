package utils

import (
	"testing"
)

func TestNormalizeGmapsEmbed(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected *string
	}{
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "Empty input",
			input:    stringPtr("   "),
			expected: nil,
		},
		{
			name: "Full iframe HTML tag",
			input: stringPtr(`<iframe src="https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d31629.301850191314!2d110.46195920000001!3d-7.719289000000001!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x2e7a5a5e301fcdfb%3A0xaa8bc699d3f127b2!2sRSU%20Mitra%20Paramedika!5e0!3m2!1sen!2sid!4v1789810367302!5m2!1sen!2sid" width="600" height="450" style="border:0;" allowfullscreen="" loading="lazy" referrerpolicy="strict-origin-when-cross-origin"></iframe>`),
			expected: stringPtr("https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d31629.301850191314!2d110.46195920000001!3d-7.719289000000001!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x2e7a5a5e301fcdfb%3A0xaa8bc699d3f127b2!2sRSU%20Mitra%20Paramedika!5e0!3m2!1sen!2sid!4v1789810367302!5m2!1sen!2sid"),
		},
		{
			name:     "Direct embed URL",
			input:    stringPtr("https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d31629.301850191314"),
			expected: stringPtr("https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d31629.301850191314"),
		},
		{
			name:     "Standard Google Maps share URL",
			input:    stringPtr("https://maps.app.goo.gl/pxDpbaZ1GTtXHTD28"),
			expected: stringPtr("https://maps.app.goo.gl/pxDpbaZ1GTtXHTD28"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeGmapsEmbed(tt.input)
			if (result == nil && tt.expected != nil) || (result != nil && tt.expected == nil) {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
			if result != nil && tt.expected != nil && *result != *tt.expected {
				t.Fatalf("expected %s, got %s", *tt.expected, *result)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
