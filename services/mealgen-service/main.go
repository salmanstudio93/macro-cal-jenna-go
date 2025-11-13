package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MacroPath/macro-path-backend/services/mealgen-service/models"
	"github.com/MacroPath/macro-path-backend/services/mealgen-service/services"
	"github.com/joho/godotenv"
)

var (
	once          sync.Once
	geminiService *services.GeminiService
	foodService   *services.FoodService
)

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func mealGenHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var reqBody models.RequestBody

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %s", err), http.StatusBadRequest)
		return
	}

	response, err := geminiService.GenerateMeals(reqBody)
	if err != nil {
		log.Printf("Error calling Gemini API: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate response: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Gemini API response received successfully")

	result := swapFoodItems(*response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func mealRegenerationHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var reqBody models.RegenerationRequest

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %s", err), http.StatusBadRequest)
		return
	}

	// Debug: Log the request data
	log.Printf("Regeneration Request - Meal: %s, Time: %s %s",
		reqBody.OriginalMeal.MealName, reqBody.OriginalMeal.MealTime, reqBody.OriginalMeal.Meridiem)
	log.Printf("Regeneration Request - Macro Targets: Calories=%.1f, Protein=%.1f, Carbs=%.1f, Fat=%.1f",
		reqBody.OriginalMeal.MacroTarget.Calories, reqBody.OriginalMeal.MacroTarget.Proteins,
		reqBody.OriginalMeal.MacroTarget.Carbs, reqBody.OriginalMeal.MacroTarget.Fats)

	// Validate request data
	if reqBody.OriginalMeal.MealName == "" {
		http.Error(w, "Invalid request: missing meal name", http.StatusBadRequest)
		return
	}
	if reqBody.OriginalMeal.MacroTarget.Calories == 0 {
		http.Error(w, "Invalid request: missing macro targets", http.StatusBadRequest)
		return
	}

	response, err := geminiService.RegenerateMeal(reqBody)
	if err != nil {
		log.Printf("Error calling Gemini API for regeneration: %v", err)
		http.Error(w, fmt.Sprintf("Failed to regenerate meal: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Gemini API regeneration response received successfully")

	result := processRegenerationResponse(*response, reqBody)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Optimized swapFoodItems with caching, better concurrency, reduced allocations, and timing tracking
func swapFoodItems(llmResponse models.MealPlanLLMResponse) models.MealPlanAPIResponse {
	// Start total timing
	totalStart := time.Now()

	result := models.MealPlanAPIResponse{
		Success:        true,
		Message:        llmResponse.Message,
		Data:           make(map[string]models.DayAPIMeals, len(llmResponse.Data)),
		Prepare:        llmResponse.Prepare,
		Cook:           llmResponse.Cook,
		WeightAssemble: llmResponse.WeightAssemble,
	}

	// Step 1: Data Collection Timing
	dataCollectionStart := time.Now()
	uniqueFoods := make(map[string]bool)
	allMeals := make([]mealProcessingData, 0)

	for key, dayMeals := range llmResponse.Data {
		for i, mealItem := range dayMeals.Meals {
			for _, foodWithPortion := range mealItem.Foods {
				uniqueFoods[foodWithPortion.Name] = true
			}
			allMeals = append(allMeals, mealProcessingData{
				dayKey:    key,
				mealIndex: i,
				mealItem:  mealItem,
				dayMeals:  dayMeals,
			})
		}
	}
	dataCollectionTime := time.Since(dataCollectionStart)

	// Step 2: Food Fetching Timing
	foodFetchingStart := time.Now()
	foodResults := batchFetchFoods(uniqueFoods)
	foodFetchingTime := time.Since(foodFetchingStart)

	// Step 3: Serving Optimization Timing
	servingOptimizationStart := time.Now()

	// Process all meals with pre-fetched food data
	for _, mealData := range allMeals {
		mealItem := mealData.mealItem
		foods := make([]models.Food, 0, len(mealItem.Foods))

		// Build foods list from pre-fetched results
		for _, foodWithPortion := range mealItem.Foods {
			if food, exists := foodResults[foodWithPortion.Name]; exists && food != nil {
				// Filter servings to keep only gram-based ones, use first as selected
				food.Servings = filterGramServings(food.Servings)
				if len(food.Servings) > 0 {
					// Ensure first serving has all required fields populated
					food.Servings[0] = ensureServingFields(food.Servings[0], food.Servings)
				}
				foods = append(foods, *food)
			}
		}

		// Select gram-based servings and adjust based on portion ratios
		optimizedFoods := adjustServingsByPortionRatio(foods, mealItem.Foods, mealItem.MacroTarget.Calories)

		// Rebalance macros to correct low fats and excess carbs while keeping realism
		optimizedFoods = rebalanceMealFoods(optimizedFoods, mealItem.MacroTarget)

		// Initialize day data if not exists
		if _, exists := result.Data[mealData.dayKey]; !exists {
			result.Data[mealData.dayKey] = models.DayAPIMeals{
				Date:  mealData.dayMeals.Date,
				Meals: make([]models.MealAPIItems, len(mealData.dayMeals.Meals)),
			}
		}

		// Calculate total macros for the meal
		totalMacros := calculateMealMacros(optimizedFoods)

		result.Data[mealData.dayKey].Meals[mealData.mealIndex] = models.MealAPIItems{
			MealName:    mealItem.MealName,
			MealTime:    mealItem.MealTime,
			Meridiem:    mealItem.Meridiem,
			MacroTarget: mealItem.MacroTarget,
			Macros:      totalMacros,
			Foods:       optimizedFoods,
		}
	}
	servingOptimizationTime := time.Since(servingOptimizationStart)

	// Step 4: Response Build Timing
	responseBuildStart := time.Now()
	totalDuration := time.Since(totalStart)
	responseBuildTime := time.Since(responseBuildStart)

	// Add timing information to response
	result.Timing = &models.TimingInfo{
		TotalDuration:       formatDuration(totalDuration),
		DataCollectionTime:  formatDuration(dataCollectionTime),
		FoodFetchingTime:    formatDuration(foodFetchingTime),
		ServingOptimization: formatDuration(servingOptimizationTime),
		ResponseBuildTime:   formatDuration(responseBuildTime),
	}

	return result
}

// processRegenerationResponse processes regeneration response and returns single meal object
func processRegenerationResponse(llmResponse models.RegenerationLLMResponse, reqBody models.RegenerationRequest) models.RegenerationResponse {
	// Start total timing
	totalStart := time.Now()

	// Step 1: Data Collection Timing
	dataCollectionStart := time.Now()
	uniqueFoods := make(map[string]bool)
	for _, foodWithPortion := range llmResponse.Data.Foods {
		uniqueFoods[foodWithPortion.Name] = true
	}
	dataCollectionTime := time.Since(dataCollectionStart)

	// Step 2: Food Fetching Timing
	foodFetchingStart := time.Now()
	foodResults := batchFetchFoods(uniqueFoods)
	foodFetchingTime := time.Since(foodFetchingStart)

	// Step 3: Serving Optimization Timing
	servingOptimizationStart := time.Now()

	// Build foods list from pre-fetched results
	foods := make([]models.Food, 0, len(llmResponse.Data.Foods))
	for _, foodWithPortion := range llmResponse.Data.Foods {
		if food, exists := foodResults[foodWithPortion.Name]; exists && food != nil {
			// Filter servings to keep only gram-based ones, use first as selected
			food.Servings = filterGramServings(food.Servings)
			if len(food.Servings) > 0 {
				// Ensure first serving has all required fields populated
				food.Servings[0] = ensureServingFields(food.Servings[0], food.Servings)
			}
			foods = append(foods, *food)
		}
	}

	// Select gram-based servings and adjust based on portion ratios
	optimizedFoods := adjustServingsByPortionRatio(foods, llmResponse.Data.Foods, llmResponse.Data.MacroTarget.Calories)

	// Rebalance macros to correct low fats and excess carbs while keeping realism
	optimizedFoods = rebalanceMealFoods(optimizedFoods, llmResponse.Data.MacroTarget)

	// Calculate total macros for the meal
	totalMacros := calculateMealMacros(optimizedFoods)

	servingOptimizationTime := time.Since(servingOptimizationStart)

	// Step 4: Response Build Timing
	responseBuildStart := time.Now()
	totalDuration := time.Since(totalStart)
	responseBuildTime := time.Since(responseBuildStart)

	// Debug: Log what we're using for the response
	log.Printf("Regeneration Response - Using Original Meal: %s, Time: %s %s",
		reqBody.OriginalMeal.MealName, reqBody.OriginalMeal.MealTime, reqBody.OriginalMeal.Meridiem)
	log.Printf("Regeneration Response - Using Original Macros: Calories=%.1f, Protein=%.1f, Carbs=%.1f, Fat=%.1f",
		reqBody.OriginalMeal.MacroTarget.Calories, reqBody.OriginalMeal.MacroTarget.Proteins,
		reqBody.OriginalMeal.MacroTarget.Carbs, reqBody.OriginalMeal.MacroTarget.Fats)
	log.Printf("Regeneration Response - Calculated Macros: Calories=%.1f, Protein=%.1f, Carbs=%.1f, Fat=%.1f",
		totalMacros.Calories, totalMacros.Proteins, totalMacros.Carbs, totalMacros.Fats)

	// Create regeneration response - always use original meal data to ensure consistency
	result := models.RegenerationResponse{
		Success:        true,
		Message:        llmResponse.Message,
		Prepare:        llmResponse.Prepare,
		Cook:           llmResponse.Cook,
		WeightAssemble: llmResponse.WeightAssemble,
		Data: models.RegenerationMealData{
			MealName:    reqBody.OriginalMeal.MealName,    // Always use original
			MealTime:    reqBody.OriginalMeal.MealTime,    // Always use original
			Meridiem:    reqBody.OriginalMeal.Meridiem,    // Always use original
			MacroTarget: reqBody.OriginalMeal.MacroTarget, // Always use original
			Macros:      totalMacros,
			Foods:       optimizedFoods,
		},
		Timing: &models.TimingInfo{
			TotalDuration:       formatDuration(totalDuration),
			DataCollectionTime:  formatDuration(dataCollectionTime),
			FoodFetchingTime:    formatDuration(foodFetchingTime),
			ServingOptimization: formatDuration(servingOptimizationTime),
			ResponseBuildTime:   formatDuration(responseBuildTime),
		},
	}

	return result
}

// formatDuration formats a duration to a readable string with appropriate precision
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000.0)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// mealProcessingData holds data for processing individual meals
type mealProcessingData struct {
	dayKey    string
	mealIndex int
	mealItem  models.MealLLMItems
	dayMeals  models.DayLLMMeals
}

// batchFetchFoods efficiently fetches all unique foods with controlled concurrency
func batchFetchFoods(uniqueFoods map[string]bool) map[string]*models.Food {
	foodResults := make(map[string]*models.Food, len(uniqueFoods))

	// Use a semaphore to limit concurrent requests (max 10 concurrent)
	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	// Track API calls
	apiCalls := 0

	for foodName := range uniqueFoods {
		apiCalls++
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Fetch food data
			searchResult, err := foodService.SearchFood(name)
			var food *models.Food
			if err == nil && len(searchResult.Foods) > 0 {
				food = &searchResult.Foods[0]
			}

			// Store result thread-safely
			mutex.Lock()
			foodResults[name] = food
			mutex.Unlock()
		}(foodName)
	}

	wg.Wait()

	// Log performance metrics
	log.Printf("Food fetching: %d API calls", apiCalls)

	return foodResults
}

// ensureServingFields ensures that the selected serving has all required fields populated
func ensureServingFields(selectedServing models.Serving, availableServings []models.Serving) models.Serving {
	// If the selected serving is empty or missing key fields, use the first available serving
	if selectedServing.ServingID == "" || selectedServing.Calories == "" {
		if len(availableServings) > 0 {
			selectedServing = availableServings[0]
		}
	}

	// Ensure all required fields are populated with default values if empty
	if selectedServing.ServingID == "" {
		selectedServing.ServingID = "default"
	}
	if selectedServing.ServingDescription == "" {
		selectedServing.ServingDescription = "1 serving"
	}
	// Only set defaults if values are completely missing, don't override food API values
	if selectedServing.MeasurementDescription == "" {
		selectedServing.MeasurementDescription = "g"
	}
	if selectedServing.MetricServingAmount == "" {
		selectedServing.MetricServingAmount = "1"
	}
	if selectedServing.MetricServingUnit == "" {
		selectedServing.MetricServingUnit = "g"
	}
	if selectedServing.NumberOfUnits == "" {
		selectedServing.NumberOfUnits = "1"
	}
	if selectedServing.Calories == "" {
		selectedServing.Calories = "0"
	}
	if selectedServing.Protein == "" {
		selectedServing.Protein = "0"
	}
	if selectedServing.Carbohydrate == "" {
		selectedServing.Carbohydrate = "0"
	}
	if selectedServing.Fat == "" {
		selectedServing.Fat = "0"
	}
	if selectedServing.Sugar == "" {
		selectedServing.Sugar = "0"
	}
	if selectedServing.Fiber == "" {
		selectedServing.Fiber = "0"
	}
	if selectedServing.SaturatedFat == "" {
		selectedServing.SaturatedFat = "0"
	}
	if selectedServing.MonounsaturatedFat == "" {
		selectedServing.MonounsaturatedFat = "0"
	}
	if selectedServing.PolyunsaturatedFat == "" {
		selectedServing.PolyunsaturatedFat = "0"
	}
	if selectedServing.Cholesterol == "" {
		selectedServing.Cholesterol = "0"
	}
	if selectedServing.Sodium == "" {
		selectedServing.Sodium = "0"
	}
	if selectedServing.Potassium == "" {
		selectedServing.Potassium = "0"
	}
	if selectedServing.Calcium == "" {
		selectedServing.Calcium = "0"
	}
	if selectedServing.Iron == "" {
		selectedServing.Iron = "0"
	}
	if selectedServing.VitaminA == "" {
		selectedServing.VitaminA = "0"
	}
	if selectedServing.VitaminB == "" {
		selectedServing.VitaminB = "0"
	}
	if selectedServing.VitaminC == "" {
		selectedServing.VitaminC = "0"
	}
	if selectedServing.VitaminD == "" {
		selectedServing.VitaminD = "0"
	}

	return selectedServing
}

// calculateMealMacros calculates the total macros for all foods in a meal
func calculateMealMacros(foods []models.Food) models.MacroTarget {
	var totalCalories, totalCarbs, totalProteins, totalFats float64

	for _, food := range foods {
		// Use first serving (which is now the selected gram-based serving)
		if len(food.Servings) > 0 {
			serving := food.Servings[0]

			// Parse and add calories
			if calories, err := strconv.ParseFloat(serving.Calories, 64); err == nil {
				totalCalories += calories
			}

			// Parse and add carbs
			if carbs, err := strconv.ParseFloat(serving.Carbohydrate, 64); err == nil {
				totalCarbs += carbs
			}

			// Parse and add protein
			if protein, err := strconv.ParseFloat(serving.Protein, 64); err == nil {
				totalProteins += protein
			}

			// Parse and add fat
			if fat, err := strconv.ParseFloat(serving.Fat, 64); err == nil {
				totalFats += fat
			}
		}
	}

	return models.MacroTarget{
		Calories: totalCalories,
		Carbs:    totalCarbs,
		Proteins: totalProteins,
		Fats:     totalFats,
	}
}

// adjustServingsByPortionRatio selects gram-based servings and adjusts them based on portion ratios
func adjustServingsByPortionRatio(foods []models.Food, foodWithPortions []models.FoodWithPortion, targetCalories float64) []models.Food {
	optimizedFoods := make([]models.Food, len(foods))

	for i, food := range foods {
		optimizedFoods[i] = food

		// Use first serving (which is now the selected gram-based serving)
		if len(food.Servings) > 0 {
			// Find the portion ratio for this food
			portionRatio := findPortionRatio(food.FoodName, foodWithPortions)

			// Calculate target calories for this food
			targetCaloriesForFood := (targetCalories * float64(portionRatio)) / 100.0

			// Adjust the first serving based on portion ratio
			adjustedServing := adjustServingForTargetCalories(food.Servings[0], targetCaloriesForFood)
			optimizedFoods[i].Servings[0] = adjustedServing
		}
	}

	return optimizedFoods
}

// findPortionRatio finds the portion ratio for a given food name
func findPortionRatio(foodName string, foodWithPortions []models.FoodWithPortion) int {
	for _, foodWithPortion := range foodWithPortions {
		if strings.EqualFold(foodName, foodWithPortion.Name) {
			return foodWithPortion.PortionRatio
		}
	}
	// Default to equal distribution if not found
	return 100 / len(foodWithPortions)
}

// adjustServingForTargetCalories adjusts a serving to match target calories
func adjustServingForTargetCalories(serving models.Serving, targetCalories float64) models.Serving {
	// Parse current calories and serving amount
	currentCalories, err := strconv.ParseFloat(serving.Calories, 64)
	if err != nil || currentCalories == 0 {
		return serving // Return original if can't parse
	}

	currentAmount, err := strconv.ParseFloat(serving.MetricServingAmount, 64)
	if err != nil || currentAmount == 0 {
		return serving // Return original if can't parse
	}

	// Calculate the multiplier needed
	multiplier := targetCalories / currentCalories

	// Adjust all nutritional values by the multiplier
	adjustedServing := serving

	// Update serving amount
	adjustedServing.MetricServingAmount = fmt.Sprintf("%.3f", currentAmount*multiplier)

	// Update all nutritional values
	adjustedServing = adjustNutritionalValue(adjustedServing, "calories", currentCalories*multiplier)
	adjustedServing = adjustNutritionalValue(adjustedServing, "protein", parseAndMultiply(serving.Protein, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "carbohydrate", parseAndMultiply(serving.Carbohydrate, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "fat", parseAndMultiply(serving.Fat, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "sugar", parseAndMultiply(serving.Sugar, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "fiber", parseAndMultiply(serving.Fiber, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "saturated_fat", parseAndMultiply(serving.SaturatedFat, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "monounsaturated_fat", parseAndMultiply(serving.MonounsaturatedFat, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "polyunsaturated_fat", parseAndMultiply(serving.PolyunsaturatedFat, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "cholesterol", parseAndMultiply(serving.Cholesterol, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "sodium", parseAndMultiply(serving.Sodium, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "potassium", parseAndMultiply(serving.Potassium, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "calcium", parseAndMultiply(serving.Calcium, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "iron", parseAndMultiply(serving.Iron, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "vitamin_a", parseAndMultiply(serving.VitaminA, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "vitamin_b", parseAndMultiply(serving.VitaminB, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "vitamin_c", parseAndMultiply(serving.VitaminC, multiplier))
	adjustedServing = adjustNutritionalValue(adjustedServing, "vitamin_d", parseAndMultiply(serving.VitaminD, multiplier))

	return adjustedServing
}

// parseAndMultiply parses a string value and multiplies it by the multiplier
func parseAndMultiply(value string, multiplier float64) float64 {
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		return parsed * multiplier
	}
	return 0
}

// adjustNutritionalValue updates a nutritional value in the serving
func adjustNutritionalValue(serving models.Serving, field string, newValue float64) models.Serving {
	valueStr := fmt.Sprintf("%.3f", newValue)

	switch field {
	case "calories":
		serving.Calories = valueStr
	case "protein":
		serving.Protein = valueStr
	case "carbohydrate":
		serving.Carbohydrate = valueStr
	case "fat":
		serving.Fat = valueStr
	case "sugar":
		serving.Sugar = valueStr
	case "fiber":
		serving.Fiber = valueStr
	case "saturated_fat":
		serving.SaturatedFat = valueStr
	case "monounsaturated_fat":
		serving.MonounsaturatedFat = valueStr
	case "polyunsaturated_fat":
		serving.PolyunsaturatedFat = valueStr
	case "cholesterol":
		serving.Cholesterol = valueStr
	case "sodium":
		serving.Sodium = valueStr
	case "potassium":
		serving.Potassium = valueStr
	case "calcium":
		serving.Calcium = valueStr
	case "iron":
		serving.Iron = valueStr
	case "vitamin_a":
		serving.VitaminA = valueStr
	case "vitamin_b":
		serving.VitaminB = valueStr
	case "vitamin_c":
		serving.VitaminC = valueStr
	case "vitamin_d":
		serving.VitaminD = valueStr
	}

	return serving
}

// filterGramServings filters servings to keep only gram-based ones
func filterGramServings(servings []models.Serving) []models.Serving {
	var gramServings []models.Serving
	for _, serving := range servings {
		// Check if this is a gram-based serving by looking at the measurement description
		descriptionLower := strings.ToLower(serving.MeasurementDescription)

		// Look for servings with gram-based descriptions
		if descriptionLower == "g" || descriptionLower == "gram" || descriptionLower == "grams" {
			gramServings = append(gramServings, serving)
		}
	}

	// If no gram servings found, return the original list (fallback to first serving)
	if len(gramServings) == 0 && len(servings) > 0 {
		return []models.Serving{servings[0]}
	}

	return gramServings
}

// findGramServing finds a gram-based serving from the available servings (deprecated - use filterGramServings)
func findGramServing(servings []models.Serving) *models.Serving {
	for _, serving := range servings {
		// Check if this is a gram-based serving by looking at the measurement description
		descriptionLower := strings.ToLower(serving.MeasurementDescription)

		// Look for servings with gram-based descriptions
		if descriptionLower == "g" || descriptionLower == "gram" || descriptionLower == "grams" {
			return &serving
		}
	}
	return nil
}

// rebalanceMealFoods adjusts servings to increase fats if under target and trim starchy carbs if over target
func rebalanceMealFoods(foods []models.Food, target models.MacroTarget) []models.Food {
	const tolerance = 0.05 // 5% tolerance

	// Run a couple of light passes to avoid drastic swings
	for pass := 0; pass < 2; pass++ {
		totals := calculateMealMacros(foods)

		// If fats are under target, try increasing a whole-food fat first
		fatLowerBound := target.Fats * (1.0 - tolerance)
		if totals.Fats < fatLowerBound {
			neededFat := fatLowerBound - totals.Fats
			// Prefer whole-food fats; fallback to higher-fat proteins if needed
			idx := findBestFatFoodIndex(foods)
			if idx >= 0 && len(foods[idx].Servings) > 0 {
				serving := foods[idx].Servings[0]
				fatPerUnit := parseFloatDefault(serving.Fat)
				if fatPerUnit > 0 {
					// Increase by a modest factor proportional to needed grams
					// Cap to avoid unrealistic portions
					factor := 1.0 + minFloat(0.6, neededFat/fatPerUnit*0.8)
					foods[idx].Servings[0] = scaleServing(serving, factor)
				}
			}
		}

		// If carbs exceed target, trim starchy carbs first
		carbUpperBound := target.Carbs * (1.0 + tolerance)
		if totals.Carbs > carbUpperBound {
			excessCarb := totals.Carbs - carbUpperBound
			starchyIndexes := findStarchyCarbIndexes(foods)
			if len(starchyIndexes) > 0 {
				// Compute total carbs from starchy sources
				var starchCarbs float64
				for _, i := range starchyIndexes {
					if len(foods[i].Servings) > 0 {
						starchCarbs += parseFloatDefault(foods[i].Servings[0].Carbohydrate)
					}
				}
				if starchCarbs > 0 {
					// Reduce starchy carbs proportionally; cap reduction per pass
					reductionFrac := minFloat(0.35, excessCarb/starchCarbs)
					factor := 1.0 - reductionFrac
					for _, i := range starchyIndexes {
						if len(foods[i].Servings) > 0 {
							foods[i].Servings[0] = scaleServing(foods[i].Servings[0], factor)
						}
					}
				}
			}
		}
	}

	return foods
}

// scaleServing multiplies serving amount and all nutrient fields by factor
func scaleServing(serving models.Serving, factor float64) models.Serving {
	if factor <= 0 {
		return serving
	}
	currentAmount := parseFloatDefault(serving.MetricServingAmount)
	serving.MetricServingAmount = fmt.Sprintf("%.3f", currentAmount*factor)

	serving.Calories = fmt.Sprintf("%.3f", parseFloatDefault(serving.Calories)*factor)
	serving.Protein = fmt.Sprintf("%.3f", parseFloatDefault(serving.Protein)*factor)
	serving.Carbohydrate = fmt.Sprintf("%.3f", parseFloatDefault(serving.Carbohydrate)*factor)
	serving.Fat = fmt.Sprintf("%.3f", parseFloatDefault(serving.Fat)*factor)
	serving.Sugar = fmt.Sprintf("%.3f", parseFloatDefault(serving.Sugar)*factor)
	serving.Fiber = fmt.Sprintf("%.3f", parseFloatDefault(serving.Fiber)*factor)
	serving.SaturatedFat = fmt.Sprintf("%.3f", parseFloatDefault(serving.SaturatedFat)*factor)
	serving.MonounsaturatedFat = fmt.Sprintf("%.3f", parseFloatDefault(serving.MonounsaturatedFat)*factor)
	serving.PolyunsaturatedFat = fmt.Sprintf("%.3f", parseFloatDefault(serving.PolyunsaturatedFat)*factor)
	serving.Cholesterol = fmt.Sprintf("%.3f", parseFloatDefault(serving.Cholesterol)*factor)
	serving.Sodium = fmt.Sprintf("%.3f", parseFloatDefault(serving.Sodium)*factor)
	serving.Potassium = fmt.Sprintf("%.3f", parseFloatDefault(serving.Potassium)*factor)
	serving.Calcium = fmt.Sprintf("%.3f", parseFloatDefault(serving.Calcium)*factor)
	serving.Iron = fmt.Sprintf("%.3f", parseFloatDefault(serving.Iron)*factor)
	serving.VitaminA = fmt.Sprintf("%.3f", parseFloatDefault(serving.VitaminA)*factor)
	serving.VitaminB = fmt.Sprintf("%.3f", parseFloatDefault(serving.VitaminB)*factor)
	serving.VitaminC = fmt.Sprintf("%.3f", parseFloatDefault(serving.VitaminC)*factor)
	serving.VitaminD = fmt.Sprintf("%.3f", parseFloatDefault(serving.VitaminD)*factor)
	return serving
}

func parseFloatDefault(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// findBestFatFoodIndex finds an index of a likely whole-food fat; prioritizes avocado, nuts, seeds, nut butters, cheese; falls back to high-fat proteins
func findBestFatFoodIndex(foods []models.Food) int {
	bestIdx := -1
	// Primary fat sources
	for i, f := range foods {
		if isWholeFoodFat(f.FoodName) {
			bestIdx = i
			break
		}
	}
	if bestIdx != -1 {
		return bestIdx
	}
	// Fallback: high-fat proteins like salmon, beef, eggs
	for i, f := range foods {
		name := strings.ToLower(f.FoodName)
		if strings.Contains(name, "salmon") || strings.Contains(name, "beef") || strings.Contains(name, "egg") || strings.Contains(name, "whole milk") || strings.Contains(name, "cheese") {
			return i
		}
	}
	return -1
}

func isWholeFoodFat(name string) bool {
	n := strings.ToLower(name)
	fatKeywords := []string{"avocado", "almond", "walnut", "pecan", "cashew", "pistachio", "hazelnut", "macadamia", "peanut", "nut butter", "peanut butter", "almond butter", "tahini", "sesame", "sunflower seed", "pumpkin seed", "chia", "flax", "hemp", "olive oil", "olives", "cheese"}
	for _, k := range fatKeywords {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func findStarchyCarbIndexes(foods []models.Food) []int {
	var idxs []int
	for i, f := range foods {
		if isStarchyCarb(f.FoodName) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func isStarchyCarb(name string) bool {
	n := strings.ToLower(name)
	starch := []string{"rice", "oat", "oatmeal", "potato", "sweet potato", "pasta", "quinoa", "bread", "tortilla", "corn", "couscous", "barley"}
	for _, k := range starch {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func generateProgramSSEHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Received SSE request from %s", r.RemoteAddr)
	enableCORS(w)

	if r.Method == "OPTIONS" {
		log.Println("✅ OPTIONS request handled")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get payload from query parameter
	payloadStr := r.URL.Query().Get("payload")
	log.Printf("📦 Payload length: %d bytes", len(payloadStr))
	if payloadStr == "" {
		log.Println("❌ Missing payload parameter")
		http.Error(w, "Missing payload parameter", http.StatusBadRequest)
		return
	}

	// Decode the payload
	var reqBody models.RequestBody
	if err := json.Unmarshal([]byte(payloadStr), &reqBody); err != nil {
		http.Error(w, fmt.Sprintf("Invalid payload JSON: %s", err), http.StatusBadRequest)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Generate the meal plan
	response, err := geminiService.GenerateMeals(reqBody)
	if err != nil {
		fmt.Fprintf(w, "data: Error: %v\n\n", err)
		flusher.Flush()
		return
	}

	result := swapFoodItems(*response)

	// Stream the data for each day
	for dayKey, dayData := range result.Data {
		// Send DAY_START marker
		fmt.Fprintf(w, "data: <DAY_START>\n\n")
		flusher.Flush()

		// Stream each meal
		for _, meal := range dayData.Meals {
			// Send MEAL_START marker
			fmt.Fprintf(w, "data: <MEAL_START>\n\n")
			flusher.Flush()

			// Create a single meal response
			mealResponse := map[string]interface{}{
				"day":   dayKey,
				"meals": []models.MealAPIItems{meal},
			}

			// Send the meal data as JSON
			mealJSON, err := json.Marshal(mealResponse)
			if err != nil {
				log.Printf("Error marshaling meal: %v", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", string(mealJSON))
			flusher.Flush()

			// Send MEAL_END marker
			fmt.Fprintf(w, "data: <MEAL_END>\n\n")
			flusher.Flush()

			// Small delay between meals for better UX
			time.Sleep(100 * time.Millisecond)
		}

		// Send DAY_END marker
		fmt.Fprintf(w, "data: <DAY_END>\n\n")
		flusher.Flush()
	}

	// Send completion marker
	fmt.Fprintf(w, "data: <MEAL_PLAN_END>\n\n")
	flusher.Flush()
}

func generateProgramSSEPostHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Received POST SSE request from %s (Method: %s)", r.RemoteAddr, r.Method)

	// Set CORS headers for all requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		log.Println("✅ OPTIONS preflight request handled")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Decode the payload from request body
	var reqBody models.RequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Printf("❌ Error decoding request body: %v", err)
		log.Printf("Content-Type: %s", r.Header.Get("Content-Type"))
		log.Printf("Content-Length: %s", r.Header.Get("Content-Length"))
		http.Error(w, fmt.Sprintf("Invalid request body: %s", err), http.StatusBadRequest)
		return
	}

	log.Printf("📦 Request decoded successfully")
	log.Printf("User: %s, Age: %d, Meals: %s, Diet: %s", reqBody.Name, reqBody.Age, reqBody.MealsPerDay, reqBody.DietType)

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Generate the meal plan
	log.Println("🔄 Calling Gemini API...")
	response, err := geminiService.GenerateMeals(reqBody)
	if err != nil {
		log.Printf("❌ Error from Gemini API: %v", err)
		fmt.Fprintf(w, "data: Error: %v\n\n", err)
		flusher.Flush()
		return
	}

	log.Println("✅ Gemini API response received")
	result := swapFoodItems(*response)

	log.Println("🚀 Starting to stream meal data...")
	// Stream the data for each day
	for dayKey, dayData := range result.Data {
		// Send DAY_START marker
		fmt.Fprintf(w, "data: <DAY_START>\n\n")
		flusher.Flush()

		// Stream each meal
		for _, meal := range dayData.Meals {
			// Send MEAL_START marker
			fmt.Fprintf(w, "data: <MEAL_START>\n\n")
			flusher.Flush()

			// Create a single meal response
			mealResponse := map[string]interface{}{
				"day":   dayKey,
				"meals": []models.MealAPIItems{meal},
			}

			// Send the meal data as JSON
			mealJSON, err := json.Marshal(mealResponse)
			if err != nil {
				log.Printf("Error marshaling meal: %v", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", string(mealJSON))
			flusher.Flush()

			// Send MEAL_END marker
			fmt.Fprintf(w, "data: <MEAL_END>\n\n")
			flusher.Flush()

			// Small delay between meals for better UX
			time.Sleep(100 * time.Millisecond)
		}

		// Send DAY_END marker
		fmt.Fprintf(w, "data: <DAY_END>\n\n")
		flusher.Flush()
	}

	// Send completion marker
	fmt.Fprintf(w, "data: <MEAL_PLAN_END>\n\n")
	flusher.Flush()
	log.Println("✅ Streaming completed")
}

func corsPreflightHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 CORS preflight request from %s", r.RemoteAddr)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
	w.WriteHeader(http.StatusOK)
	log.Println("✅ CORS preflight response sent")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "mealgen-service endpoint")
}

func init() {
	once.Do(func() {
		log.Println("Initializing mealgen-service...")

		// Load environment variables from .env file
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: Failed to load .env file: %v (this is normal for Cloud Run)", err)
		}

		// Get API keys from environment variables
		geminiApiKey := os.Getenv("GEMINI_API_KEY")
		foodApiKey := os.Getenv("FOOD_API_KEY")

		// Validate required API keys
		if geminiApiKey == "" {
			log.Fatal("❌ GEMINI_API_KEY is required! Please add it to your .env file or set as environment variable")
		}
		if foodApiKey == "" {
			log.Fatal("❌ FOOD_API_KEY is required! Please add it to your .env file or set as environment variable")
		}

		log.Println("Environment variables validated successfully")

		foodService = services.NewFoodService(foodApiKey)
		geminiService = services.NewGeminiService(geminiApiKey, foodService)

		log.Println("Services initialized successfully")
		log.Println("Ready to accept requests")
	})
}

func main() {
	// Get port from environment variable, default to 8080 for local development
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /", rootHandler)
	mux.HandleFunc("OPTIONS /", corsPreflightHandler)
	mux.HandleFunc("POST /", mealGenHandler)
	mux.HandleFunc("OPTIONS /regenerate", corsPreflightHandler)
	mux.HandleFunc("POST /regenerate", mealRegenerationHandler)
	mux.HandleFunc("GET /program/generate-program", generateProgramSSEHandler)
	mux.HandleFunc("OPTIONS /program/generate-program", corsPreflightHandler)
	mux.HandleFunc("POST /program/generate-program", generateProgramSSEPostHandler)

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
