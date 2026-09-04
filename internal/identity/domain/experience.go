package domain

// ExperienceLevel is an auto-derived hiker seniority bucket (data-model →
// users → experience_level). The bucket RULES live in the achievement context
// (which owns derived badge/level math, plan.md context map). Identity declares
// only the port so its application/HTTP layers can render a profile without
// importing achievement; the composition root wires the concrete provider.
type ExperienceLevel string

const (
	LevelPemula   ExperienceLevel = "Pemula"
	LevelMenengah ExperienceLevel = "Menengah"
	LevelLanjut   ExperienceLevel = "Lanjut"
	LevelAhli     ExperienceLevel = "Ahli"
)

// ExperienceProvider derives a user's experience level from their verified
// distinct-mountain count (cross-context read port: achievement → journey).
type ExperienceProvider interface {
	// LevelFor returns the bucket for a verified distinct-mountain count.
	LevelFor(count int) ExperienceLevel
}

// LevelProviderFunc adapts a function to ExperienceProvider.
type LevelProviderFunc func(count int) ExperienceLevel

// LevelFor implements ExperienceProvider.
func (f LevelProviderFunc) LevelFor(count int) ExperienceLevel { return f(count) }
