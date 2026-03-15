package core

import (
	"strings"
	"testing"
)

func TestNormalizeBuildFailureMessage_ExplainsBuildkitPrivilegeErrors(t *testing.T) {
	raw := "buildctl: Error"
	logTail := `error: failed to solve: failed to read dockerfile: failed to mount /tmp/buildkit-mount1111380470: [{Type:bind Source:/var/lib/buildkit/runc-native/snapshots/snapshots/1 Target: Options:[rbind ro]}]: mount source: "/var/lib/buildkit/runc-native/snapshots/snapshots/1", target: "/tmp/buildkit-mount1111380470", fstype: bind, flags: 20481, data: "", err: operation not permitted`

	msg := normalizeBuildFailureMessage(raw, logTail)

	if !strings.Contains(msg, "BuildKit builder is missing required mount privileges") {
		t.Fatalf("expected privileged mount guidance, got %q", msg)
	}
	if !strings.Contains(msg, "ketches-buildkitd") {
		t.Fatalf("expected buildkitd reference, got %q", msg)
	}
}

func TestNormalizeBuildFailureMessage_ExplainsCrossArchExecutionFailures(t *testing.T) {
	raw := "buildctl: Error"
	logTail := `#19 0.210 exec /bin/sh: exec format error`

	msg := normalizeBuildFailureMessage(raw, logTail)

	if !strings.Contains(msg, "Multi-arch build requires binfmt/QEMU support") {
		t.Fatalf("expected binfmt guidance, got %q", msg)
	}
	if !strings.Contains(msg, "ketches-buildkit-binfmt") {
		t.Fatalf("expected binfmt daemonset reference, got %q", msg)
	}
}
