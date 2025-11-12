package models

type RequestBody struct {
	// User Profile
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Gender string `json:"gender"`
	Weight int    `json:"weight"`
	Height int    `json:"height"`
	Goal   string `json:"goal"`

	// Daily Macro Goals
	DailyProtiensGoal float64 `json:"DailyProtiensGoal"`
	DailyCarbsGoal    float64 `json:"DailyCarbsGoal"`
	DailyFatsGoal     float64 `json:"DailyFatsGoal"`
	DailyCaloriesGoal float64 `json:"DailyCaloriesGoal"`

	// Activity and Diet
	ActivityLevel string `json:"activity_level"`
	DietType      string `json:"diet_type"`

	// Food Preferences
	FoodAllergies []string `json:"food_allergies"`
	FoodLikes     []string `json:"food_likes"`

	// Health Information
	SelectedLifeStages       []string `json:"selectedLifeStages"`
	SelectedHealthConditions []string `json:"selectedHealthConditions"`

	// Meal Planning
	EatingWindow        string   `json:"eating_window"`
	MealsPerDay         string   `json:"meals_per_day"`
	PreferredMealTimes  string   `json:"preferred_meal_times"`
	MacroPreference     string   `json:"macro_preference"`
	CaloricIntake       int      `json:"caloric_intake"`
	GroceryAvailability string   `json:"grocery_availability"`
	Supplements         []string `json:"supplements"`

	// Meal Schedule - Dynamic meal names and times (optional)
	MealSchedule []MealScheduleItem `json:"meal_schedule,omitempty"`

	// Optional fields for backward compatibility
	Dates         []string `json:"dates,omitempty"`
	NumberOfMeals int      `json:"number_of_meals,omitempty"`
}

// MealScheduleItem represents a single meal's schedule information
type MealScheduleItem struct {
	Name     string `json:"name"`
	Time     string `json:"time"`
	Meridiem string `json:"meridiem"`
}

type MealOption struct {
	IsSelected bool        `json:"is_selected"`
	Entries    []MealEntry `json:"entries"`
}

type MealEntry struct {
	Date     string `json:"date"`
	MealName string `json:"meal_name"`
}

type DrinkEntry struct {
	Date           string `json:"date"`
	MealName       string `json:"meal_name"`
	NumberOfDrinks string `json:"number_of_drinks"`
}

type Cuisine struct {
	Name       string `json:"name"`
	Preference string `json:"preference"`
}

type MacroTarget struct {
	Calories float64 `json:"calories"`
	Carbs    float64 `json:"carbs"`
	Fats     float64 `json:"fats"`
	Proteins float64 `json:"proteins"`
}

// Response models
type MealPlanLLMResponse struct {
	Success        bool                    `json:"success"`
	Data           map[string]DayLLMMeals  `json:"data"`
	Message        string                  `json:"message,omitempty"`
	Prepare        []PrepareCookSection    `json:"prepare,omitempty"`
	Cook           []PrepareCookSection    `json:"cook,omitempty"`
	WeightAssemble []WeightAssembleSection `json:"weight_assemble,omitempty"`
}

type DayLLMMeals struct {
	Date  string         `json:"date"`
	Meals []MealLLMItems `json:"meals"`
}

type MealLLMItems struct {
	MealName       string                  `json:"meal_name"`
	MealTime       string                  `json:"meal_time"`
	Meridiem       string                  `json:"meridiem"`
	MacroTarget    MacroTarget             `json:"macro_target"`
	Foods          []FoodWithPortion       `json:"foods"`
	Prepare        []PrepareCookSection    `json:"prepare,omitempty"`
	Cook           []PrepareCookSection    `json:"cook,omitempty"`
	WeightAssemble []WeightAssembleSection `json:"weight_assemble,omitempty"`
}

type FoodWithPortion struct {
	Name         string `json:"name"`
	PortionRatio int    `json:"portion_ratio"`
}

type MealPlanAPIResponse struct {
	Success        bool                    `json:"success"`
	Data           map[string]DayAPIMeals  `json:"data"`
	Message        string                  `json:"message,omitempty"`
	Timing         *TimingInfo             `json:"timing,omitempty"`
	Prepare        []PrepareCookSection    `json:"prepare,omitempty"`
	Cook           []PrepareCookSection    `json:"cook,omitempty"`
	WeightAssemble []WeightAssembleSection `json:"weight_assemble,omitempty"`
}

// TimingInfo contains timing information for different steps
type TimingInfo struct {
	TotalDuration       string `json:"total_duration"`
	DataCollectionTime  string `json:"data_collection_time"`
	FoodFetchingTime    string `json:"food_fetching_time"`
	ServingOptimization string `json:"serving_optimization_time"`
	ResponseBuildTime   string `json:"response_build_time"`
}

type DayAPIMeals struct {
	Date  string         `json:"date"`
	Meals []MealAPIItems `json:"meals"`
}

type MealAPIItems struct {
	MealName       string                  `json:"meal_name"`
	MealTime       string                  `json:"meal_time"`
	Meridiem       string                  `json:"meridiem"`
	MacroTarget    MacroTarget             `json:"macro_target"`
	Macros         MacroTarget             `json:"macros"`
	Foods          []Food                  `json:"foods"`
	Prepare        []PrepareCookSection    `json:"prepare,omitempty"`
	Cook           []PrepareCookSection    `json:"cook,omitempty"`
	WeightAssemble []WeightAssembleSection `json:"weight_assemble,omitempty"`
}

// Meal Preferences Models for serving selection
type MealPreferencesRequest struct {
	Meals []MealWithFoods `json:"meals"`
}

type MealWithFoods struct {
	MealName    string      `json:"meal_name"`
	MealTime    string      `json:"meal_time"`
	Meridiem    string      `json:"meridiem"`
	MacroTarget MacroTarget `json:"macro_target"`
	Foods       []Food      `json:"foods"`
}

type MealPreferencesResponse struct {
	Success bool                                `json:"success"`
	Data    map[string]MealWithSelectedServings `json:"data"`
	Message string                              `json:"message,omitempty"`
}

type MealWithSelectedServings struct {
	MealName string                    `json:"meal_name"`
	MealTime string                    `json:"meal_time"`
	Meridiem string                    `json:"meridiem"`
	Foods    []FoodWithSelectedServing `json:"foods"`
}

type FoodWithSelectedServing struct {
	FoodID   string `json:"food_id"`
	FoodName string `json:"food_name"`
}

// Simplified response models for LLM serving selection
type ServingSelectionRequest struct {
	Meals []MealWithFoods `json:"meals"`
}

type ServingSelectionResponse struct {
	Success bool                          `json:"success"`
	Data    map[string]MealServingChoices `json:"data"`
	Message string                        `json:"message,omitempty"`
}

type MealServingChoices struct {
	MealName string              `json:"meal_name"`
	MealTime string              `json:"meal_time"`
	Meridiem string              `json:"meridiem"`
	Foods    []FoodServingChoice `json:"foods"`
}

type FoodServingChoice struct {
	FoodID               string  `json:"food_id"`
	FoodName             string  `json:"food_name"`
	PercentageAdjustment float64 `json:"percentage_adjustment"`
}

// New structured models for prepare, cook, and weight & assemble
type PrepareCookSection struct {
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Steps    []string `json:"steps"`
}

type WeightAssembleSection struct {
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Steps    []string `json:"steps"`
}
