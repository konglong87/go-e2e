package computeruse

import (
	"testing"
)

func TestStaticTargetRegistryResolvesTrustedMetadata(t *testing.T) {
	registry, err := NewStaticTargetRegistry(ApplicationTarget{
		ID:          "calculator",
		DisplayName: "Calculator",
		Launch:      LaunchPolicy{ProviderKey: "com.apple.calculator"},
		Window:      WindowPolicy{BundleID: "com.apple.calculator", RequireVisible: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := registry.Resolve("calculator")
	if err != nil {
		t.Fatal(err)
	}
	if target.Launch.ProviderKey != "com.apple.calculator" || target.Window.BundleID != "com.apple.calculator" {
		t.Fatalf("resolved target=%+v", target)
	}
	if _, err := registry.Resolve("com.apple.safari"); err == nil {
		t.Fatal("unregistered target resolved")
	}
}

func TestStaticTargetRegistryRejectsInvalidOrDuplicateTargets(t *testing.T) {
	if _, err := NewStaticTargetRegistry(ApplicationTarget{ID: "calculator"}); err == nil {
		t.Fatal("invalid target accepted")
	}
	registry, err := NewStaticTargetRegistry(ApplicationTarget{
		ID: "editor", DisplayName: "Editor", Launch: LaunchPolicy{ProviderKey: "com.example.editor"}, Window: WindowPolicy{BundleID: "com.example.editor"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ApplicationTarget{
		ID: "editor", DisplayName: "Editor 2", Launch: LaunchPolicy{ProviderKey: "com.example.editor2"}, Window: WindowPolicy{BundleID: "com.example.editor2"},
	}); err == nil {
		t.Fatal("duplicate target accepted")
	}
}
