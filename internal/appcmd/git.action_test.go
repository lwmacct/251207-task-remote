package appcmd

import (
	"regexp"
	"testing"
)

func TestTopicBranchName(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "single name", args: []string{"fix-login"}, want: "topic/fix-login"},
		{name: "multiple words", args: []string{"fix", "login"}, want: "topic/fix-login"},
		{name: "quoted words", args: []string{"fix login"}, want: "topic/fix-login"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := topicBranchName(test.args); got != test.want {
				t.Fatalf("topicBranchName(%q) = %q, want %q", test.args, got, test.want)
			}
		})
	}
}

func TestTopicBranchNameUsesTimestampByDefault(t *testing.T) {
	got := topicBranchName(nil)
	if !regexp.MustCompile(`^topic/[0-9]{6}-[0-9]{4}$`).MatchString(got) {
		t.Fatalf("topicBranchName(nil) = %q", got)
	}
}
