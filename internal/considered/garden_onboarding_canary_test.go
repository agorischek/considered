package considered

import "testing"

func TestGardenOnboardingMustNeverMerge(t *testing.T) {
 t.Fatal("Intentional negative onboarding canary: required checks must prevent merge")
}
