package models

// Regeneration request models
type RegenerationRequest struct {
	FoodsToRegenerate []string     `json:"food_to_regenerate"` // Foods to replace (empty = regenerate entire meal)
	MealStyle         string       `json:"meal_style_option"`
	DietType          string       `json:"diet_type"`
	FoodsToAvoid      []string     `json:"foods_to_avoid"`
	FoodsToLike       []string     `json:"foods_to_like"`
	OriginalMeal      OriginalMeal `json:"meal"` // Changed from "original_meal" to "meal"

	// Additional fields that may be present but not used in regeneration
	TreatMeals   interface{} `json:"treat_meals,omitempty"`
	SocialMeals  interface{} `json:"social_meals,omitempty"`
	IsDrinks     bool        `json:"is_drinks,omitempty"`
	DrinkEntries interface{} `json:"drink_entries,omitempty"`
	Cuisines     []string    `json:"cuisines,omitempty"`
	KitchenTools []string    `json:"kitchen_tools,omitempty"`
	LifePhases   []string    `json:"life_phases,omitempty"`
}

type OriginalMeal struct {
	MealName    string      `json:"meal_name"`
	MealTime    string      `json:"meal_time"`
	Meridiem    string      `json:"meridiem"`
	MacroTarget MacroTarget `json:"macro_target"`
	Macros      MacroTarget `json:"macros"` // Current calculated macros
	Foods       []Food      `json:"foods"`
}

// Regeneration response models
type RegenerationResponse struct {
	Success        bool                    `json:"success"`
	Data           RegenerationMealData    `json:"data"`
	Message        string                  `json:"message,omitempty"`
	Timing         *TimingInfo             `json:"timing,omitempty"`
	Prepare        []PrepareCookSection    `json:"prepare,omitempty"`
	Cook           []PrepareCookSection    `json:"cook,omitempty"`
	WeightAssemble []WeightAssembleSection `json:"weight_assemble,omitempty"`
}

type RegenerationMealData struct {
	MealName    string      `json:"meal_name"`
	MealTime    string      `json:"meal_time"`
	Meridiem    string      `json:"meridiem"`
	MacroTarget MacroTarget `json:"macro_target"`
	Macros      MacroTarget `json:"macros"`
	Foods       []Food      `json:"foods"`
}

// Internal LLM response models for regeneration
type RegenerationLLMResponse struct {
	Success        bool                    `json:"success"`
	Message        string                  `json:"message"`
	Data           RegenerationLLMData     `json:"data"`
	Prepare        []PrepareCookSection    `json:"prepare,omitempty"`
	Cook           []PrepareCookSection    `json:"cook,omitempty"`
	WeightAssemble []WeightAssembleSection `json:"weight_assemble,omitempty"`
}

type RegenerationLLMData struct {
	MealName    string            `json:"meal_name"`
	MealTime    string            `json:"meal_time"`
	Meridiem    string            `json:"meridiem"`
	MacroTarget MacroTarget       `json:"macro_target"`
	Foods       []FoodWithPortion `json:"foods"`
}
