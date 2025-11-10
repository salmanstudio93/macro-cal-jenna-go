package models

type FoodAPIResult struct {
	ProviderName string `json:"provider_name"`
	SearchTag    string `json:"search_tag"`
	PageNumber   string `json:"page_number"`   //Page numbers offset, starting from 0
	MaxResults   string `json:"max_results"`   //Total results fetched
	TotalResults string `json:"total_results"` //Total available results
	Foods        []Food `json:"foods"`
}

type Food struct {
	FoodID    string    `json:"food_id"`
	FoodName  string    `json:"food_name"`
	FoodType  string    `json:"food_type"`
	BrandName string    `json:"brand_name"`
	Servings  []Serving `json:"servings"`
}

type Serving struct {
	ServingID              string `json:"serving_id"`
	ServingDescription     string `json:"serving_description"`
	MeasurementDescription string `json:"measurement_description"`
	MetricServingAmount    string `json:"metric_serving_amount"`
	MetricServingUnit      string `json:"metric_serving_unit"`
	NumberOfUnits          string `json:"number_of_units"`

	// Macro Nutrients
	Calories     string `json:"calories"`
	Protein      string `json:"protein"`
	Carbohydrate string `json:"carbohydrate"`
	Fat          string `json:"fat"`
	Sugar        string `json:"sugar"`
	Fiber        string `json:"fiber"`

	// Fats breakdown
	SaturatedFat       string `json:"saturated_fat"`
	MonounsaturatedFat string `json:"monounsaturated_fat"`
	PolyunsaturatedFat string `json:"polyunsaturated_fat"`
	Cholesterol        string `json:"cholesterol"`

	// Minerals
	Sodium    string `json:"sodium"`
	Potassium string `json:"potassium"`
	Calcium   string `json:"calcium"`
	Iron      string `json:"iron"`

	// Vitamins
	VitaminA string `json:"vitamin_a"`
	VitaminB string `json:"vitamin_b"`
	VitaminC string `json:"vitamin_c"`
	VitaminD string `json:"vitamin_d"`
}
