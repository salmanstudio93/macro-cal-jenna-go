package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/MacroPath/macro-path-backend/services/mealgen-service/models"
)

type GeminiService struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	foodService *FoodService
}

type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content Content `json:"content"`
}

func NewGeminiService(apiKey string, foodService *FoodService) *GeminiService {
	return &GeminiService{
		apiKey:      apiKey,
		baseURL:     "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent",
		client:      &http.Client{},
		foodService: foodService,
	}
}

func (gs *GeminiService) GenerateMeals(reqBody models.RequestBody) (*models.MealPlanLLMResponse, error) {
	prompt := gs.buildMealPrompt(reqBody)
	response, err := gs.prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling Gemini API: %v", err)
	}
	return gs.parseMealResponse(response, reqBody)
}

func (gs *GeminiService) GenerateSingleMeal(reqBody models.RequestBody, day string, mealName string, mealTime string, meridiem string, previousMeals []models.MealLLMItems) (*models.MealLLMItems, error) {
	prompt := gs.buildSingleMealPrompt(reqBody, day, mealName, mealTime, meridiem, previousMeals)
	response, err := gs.prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling Gemini API: %v", err)
	}
	return gs.parseSingleMealResponse(response, reqBody, day, mealName, mealTime, meridiem)
}

func (gs *GeminiService) RegenerateMeal(reqBody models.RegenerationRequest) (*models.RegenerationLLMResponse, error) {
	prompt := gs.buildRegenerationPrompt(reqBody)
	response, err := gs.prompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling Gemini API for regeneration: %v", err)
	}
	return gs.parseRegenerationResponse(response, reqBody)
}

func (gs *GeminiService) buildMealPrompt(reqBody models.RequestBody) string {
	prompt := "You are a professional nutritionist and meal planning expert. Create a comprehensive meal plan based on the user's requirements.\n\n"

	// Generate dates if not provided (7 days from today)
	dates := reqBody.Dates
	if len(dates) == 0 {
		for i := 0; i < 7; i++ {
			date := fmt.Sprintf("Day %d", i+1)
			dates = append(dates, date)
		}
	}

	// Parse meals per day
	mealsPerDay := 3 // default
	if reqBody.MealsPerDay != "" {
		if parsed, err := strconv.Atoi(reqBody.MealsPerDay); err == nil {
			mealsPerDay = parsed
		}
	}
	if reqBody.NumberOfMeals > 0 {
		mealsPerDay = reqBody.NumberOfMeals
	}

	prompt += "USER PROFILE:\n"
	if reqBody.Name != "" {
		prompt += fmt.Sprintf("- Name: %s\n", reqBody.Name)
	}
	if reqBody.Age > 0 {
		prompt += fmt.Sprintf("- Age: %d years\n", reqBody.Age)
	}
	if reqBody.Gender != "" {
		prompt += fmt.Sprintf("- Gender: %s\n", reqBody.Gender)
	}
	if reqBody.Weight > 0 {
		prompt += fmt.Sprintf("- Weight: %d kg\n", reqBody.Weight)
	}
	if reqBody.Height > 0 {
		prompt += fmt.Sprintf("- Height: %d cm\n", reqBody.Height)
	}
	if reqBody.Goal != "" {
		prompt += fmt.Sprintf("- Goal: %s\n", reqBody.Goal)
	}
	if reqBody.ActivityLevel != "" {
		prompt += fmt.Sprintf("- Activity Level: %s\n", reqBody.ActivityLevel)
	}
	prompt += "\n"

	prompt += "MEAL PLANNING REQUIREMENTS:\n"
	prompt += fmt.Sprintf("- Number of Days: %d\n", len(dates))
	prompt += fmt.Sprintf("- Diet Type: %s\n", reqBody.DietType)
	prompt += fmt.Sprintf("- Number of Meals per Day: %d\n", mealsPerDay)
	if reqBody.PreferredMealTimes != "" {
		prompt += fmt.Sprintf("- Preferred Meal Times: %s\n", reqBody.PreferredMealTimes)
	}
	if reqBody.EatingWindow != "" {
		prompt += fmt.Sprintf("- Eating Window: %s\n", reqBody.EatingWindow)
	}
	if reqBody.GroceryAvailability != "" {
		prompt += fmt.Sprintf("- Grocery Availability: %s\n", reqBody.GroceryAvailability)
	}
	prompt += "\n"

	prompt += "MACRO TARGETS:\n"
	prompt += fmt.Sprintf("- Daily Calories: %.1f\n", reqBody.DailyCaloriesGoal)
	prompt += fmt.Sprintf("- Daily Protein: %.1fg\n", reqBody.DailyProtiensGoal)
	prompt += fmt.Sprintf("- Daily Carbs: %.1fg\n", reqBody.DailyCarbsGoal)
	prompt += fmt.Sprintf("- Daily Fats: %.1fg\n", reqBody.DailyFatsGoal)
	prompt += fmt.Sprintf("- Per-Meal Targets: Calories: %.1f, Protein: %.1fg, Carbs: %.1fg, Fat: %.1fg\n",
		reqBody.DailyCaloriesGoal/float64(mealsPerDay),
		reqBody.DailyProtiensGoal/float64(mealsPerDay),
		reqBody.DailyCarbsGoal/float64(mealsPerDay),
		reqBody.DailyFatsGoal/float64(mealsPerDay))
	prompt += "\n"

	if len(reqBody.FoodAllergies) > 0 {
		prompt += fmt.Sprintf("ALLERGIES/FOODS TO AVOID: %s\n\n", strings.Join(reqBody.FoodAllergies, ", "))
	}

	if len(reqBody.FoodLikes) > 0 {
		prompt += fmt.Sprintf("FOOD PREFERENCES (LIKES): %s\n\n", strings.Join(reqBody.FoodLikes, ", "))
	}

	if len(reqBody.SelectedLifeStages) > 0 {
		prompt += fmt.Sprintf("LIFE STAGES: %s\n\n", strings.Join(reqBody.SelectedLifeStages, ", "))
	}

	if len(reqBody.SelectedHealthConditions) > 0 {
		prompt += fmt.Sprintf("HEALTH CONDITIONS: %s\n\n", strings.Join(reqBody.SelectedHealthConditions, ", "))
	}

	if len(reqBody.Supplements) > 0 {
		prompt += fmt.Sprintf("SUPPLEMENTS: %s\n\n", strings.Join(reqBody.Supplements, ", "))
	}

	prompt += "\nTASK:\n"
	prompt += fmt.Sprintf("Create a meal plan for %d days with %d meals per day.\n", len(dates), mealsPerDay)
	prompt += "Each meal should include foods that align with the user's diet type and goals.\n"
	prompt += "There is NO restriction on the number of food items - use as many foods as needed to fulfill the macro targets.\n"
	prompt += "For each food, specify the portion ratio (percentage) it should represent in the meal.\n"
	prompt += "CRITICAL: The portion ratios should be calculated to help achieve the per-meal macro targets.\n"
	prompt += "Focus on whole, unprocessed foods that provide balanced nutrition.\n"
	prompt += "RESTRICT MULTI-INGREDIENT FOODS: Avoid foods with multiple ingredients (processed foods, packaged items, complex recipes). Use single-ingredient whole foods instead.\n\n"

	prompt += "MEAL GENERATION RULES:\n"
	prompt += "1. UNIVERSAL MEAL STRUCTURE (4-Component Rule):\n"
	prompt += "   - Component 1: Protein Source (chicken, fish, beef, turkey, eggs, Greek yogurt, tofu)\n"
	prompt += "   - Component 2: Starchy Carbohydrate (50% of meal carbs) - rice, oats, potatoes, sweet potatoes, pasta, quinoa, bread, corn\n"
	prompt += "   - Component 3: Fruit or Vegetable (50% of meal carbs) - berries, apples, bananas, broccoli, peppers, spinach, mixed greens, carrots, tomatoes\n"
	prompt += "   - Component 4: Fat Source (whole-food priority: avocado, nuts, seeds, nut butters, cheese)\n\n"

	prompt += "2. MACRO DISTRIBUTION:\n"
	prompt += "   - Daily: 40% Carbs | 30% Protein | 30% Fat (fat target MUST be met)\n"
	prompt += "   - Per-Meal: Divide daily targets by number of meals\n"
	prompt += "   - CRITICAL: Split carbs 50% starchy / 50% fruit-vegetable\n"
	prompt += "   - If fat is under target after protein/carb planning, add a whole-food fat component to reach the fat target.\n\n"

	prompt += "3. HIERARCHICAL PLANNING:\n"
	prompt += "   - STEP 1: Plan carbohydrate sources first (50/50 split)\n"
	prompt += "   - STEP 2: Plan protein sources second\n"
	prompt += "   - STEP 3: Complete with fat source if needed (always include a fat component)\n\n"

	prompt += "4. BREAKFAST FOODS (for breakfast meals only):\n"
	prompt += "   - Eggs, dairy (Greek yogurt, cottage cheese, milk, cheese)\n"
	prompt += "   - Grains: Oats, cereals, granola, whole wheat bread, English muffins\n"
	prompt += "   - Proteins: Turkey bacon, Canadian bacon, breakfast sausage\n"
	prompt += "   - Fruits: Any fruits (berries, bananas, apples, etc.)\n"
	prompt += "   - Other: Avocado, nut butters, nuts, seeds, protein powder\n\n"

	prompt += "5. PORTION SPECIFICATIONS:\n"
	prompt += "   - ALL portions MUST be in GRAMS ONLY (never cups, ounces, tablespoons)\n"
	prompt += "   - Specify (cooked) or (raw) for meats, grains, starchy vegetables\n"
	prompt += "   - Examples: '150g chicken breast (cooked)', '185g brown rice (cooked)', '200g sweet potato (raw)'\n\n"

	prompt += "6. DIETARY RESTRICTIONS:\n"
	prompt += "   - Vegetarian: No meat or fish\n"
	prompt += "   - Vegan: No animal products (meat, fish, dairy, eggs)\n"
	prompt += "   - Pescatarian: Fish only, no other meat\n"
	prompt += "   - Paleo: Whole foods, no grains, dairy, or legumes\n"
	prompt += "   - Gluten-Free: No wheat, barley, rye\n"
	prompt += "   - Dairy-Free: No milk products\n\n"

	prompt += "7. CRITICAL RULES:\n"
	prompt += "   - 50/50 Carb Split: ALWAYS split carbs 50% starchy / 50% fruit-vegetable\n"
	prompt += "   - Whole-Food Fat Priority: Use nuts, seeds, avocado, nut butters BEFORE oils\n"
	prompt += "   - NO OILS OR CONDIMENTS: DO NOT include any oils (olive oil, vegetable oil, coconut oil, etc.) or condiments (ketchup, mustard, mayonnaise, etc.) in meals\n"
	prompt += "   - RESTRICT MULTI-INGREDIENT FOODS: Avoid foods with multiple ingredients (processed foods, packaged items, complex recipes). Use single-ingredient whole foods only\n"
	prompt += "   - NO FOOD COUNT RESTRICTION: Use as many food items as needed to fulfill macro targets - there is no limit on the number of foods per meal\n"
	prompt += "   - Protein/Fat Balancing: If high-fat protein reaches fat limit before protein target, add low-fat protein source\n"
	prompt += "   - Breakfast Foods Enforcement: Breakfast meals = breakfast foods ONLY\n"
	prompt += "   - Grams Only: All portions in grams (never cups, oz, tbsp)\n"
	prompt += "   - Cooked/Raw Required: All meats, grains, starchy veggies MUST specify cooked/raw\n\n"

	prompt += "VARIETY & REALISM:\n"
	prompt += "- Do not repeat the exact same food within the same day.\n"
	prompt += "- Avoid repeating the same primary protein for the same meal name on consecutive days.\n"
	prompt += "- Use realistic combinations from different cuisines across the week.\n\n"

	prompt += "PREPARE, COOK & WEIGHT & ASSEMBLE STEPS:\n"
	prompt += "For each day, provide comprehensive preparation, cooking, and assembly instructions that cover ALL meals for that day.\n"
	prompt += "These should be practical, batch-cooking focused instructions that help users efficiently prepare their meals.\n\n"

	prompt += "STRUCTURED FORMAT REQUIREMENTS:\n"
	prompt += "Each section (prepare, cook, weight_assemble) must be an array of objects with:\n"
	prompt += "- title: The main category name\n"
	prompt += "- subtitle: Optional descriptive subtitle\n"
	prompt += "- steps: Array of bullet points (not complete sentences)\n\n"

	prompt += "PREPARE section should include these categories:\n"
	prompt += "1. 'Preparing Protein' - seasoning, batch cooking methods, storage tips\n"
	prompt += "2. 'Preparing Carbs' - batch cooking grains, potatoes, etc.\n"
	prompt += "3. 'Preparing Fat' - whole-food fat sources, portioning\n\n"

	prompt += "COOK section should include these categories:\n"
	prompt += "1. 'Cook Protein' - temperatures, times, batch methods\n"
	prompt += "2. 'Cook Carbs' - rice cooker, oven, air fryer instructions\n"
	prompt += "3. 'Cook Fat' - minimal cooking needed, mostly assembly\n\n"

	prompt += "WEIGHT & ASSEMBLE section should include these categories:\n"
	prompt += "1. 'Food Scale Basics' with subtitle 'Why GRAMS (not servings/oz)'\n"
	prompt += "2. 'Food Weight vs. Macro Grams' - explain the difference\n"
	prompt += "3. 'How to Use a Food Scale' - step-by-step instructions\n"
	prompt += "4. 'Assemble Your Meals' - assembly templates and methods\n"
	prompt += "5. 'Storage' - meal prep containers, freezing tips\n\n"

	prompt += "BULLET POINT FORMAT:\n"
	prompt += "- Each step should be a concise bullet point, not a complete sentence\n"
	prompt += "- Focus on actionable instructions\n"
	prompt += "- Use simple, clear language\n"
	prompt += "- Keep each point under 20 words when possible\n\n"

	prompt += "RESPONSE FORMAT:\n"
	prompt += "Return ONLY a valid JSON object in this exact structure:\n"
	prompt += "{\n"
	prompt += "  \"success\": true,\n"
	prompt += "  \"message\": \"Meal plan created successfully\",\n"
	prompt += "  \"data\": {\n"

	// Add dates to the prompt
	if len(dates) > 0 {
		for i, date := range dates {
			if i > 0 {
				prompt += ",\n"
			}
			prompt += fmt.Sprintf("    \"%s\": {\n", date)
			prompt += fmt.Sprintf("      \"date\": \"%s\",\n", date)
		}
	} else {
		// Default example if no dates provided
		prompt += "    \"2024-01-01\": {\n"
		prompt += "      \"date\": \"2024-01-01\",\n"
	}
	prompt += "      \"meals\": [\n"
	prompt += "        {\n"
	prompt += "          \"meal_name\": \"Breakfast\",\n"
	prompt += "          \"meal_time\": \"08:00\",\n"
	prompt += "          \"meridiem\": \"AM\",\n"
	prompt += "          \"foods\": [\n"
	prompt += "            {\"name\": \"Oatmeal\", \"portion_ratio\": 40},\n"
	prompt += "            {\"name\": \"Greek Yogurt\", \"portion_ratio\": 25},\n"
	prompt += "            {\"name\": \"Banana\", \"portion_ratio\": 20},\n"
	prompt += "            {\"name\": \"Almonds\", \"portion_ratio\": 15}\n"
	prompt += "          ]\n"
	prompt += "        },\n"
	prompt += "        {\n"
	prompt += "          \"meal_name\": \"Lunch\",\n"
	prompt += "          \"meal_time\": \"13:00\",\n"
	prompt += "          \"meridiem\": \"PM\",\n"
	prompt += "          \"foods\": [\n"
	prompt += "            {\"name\": \"Grilled Chicken Breast\", \"portion_ratio\": 40},\n"
	prompt += "            {\"name\": \"Brown Rice\", \"portion_ratio\": 30},\n"
	prompt += "            {\"name\": \"Broccoli\", \"portion_ratio\": 15},\n"
	prompt += "            {\"name\": \"Avocado\", \"portion_ratio\": 15}\n"
	prompt += "          ]\n"
	prompt += "        },\n"
	prompt += "        {\n"
	prompt += "          \"meal_name\": \"Dinner\",\n"
	prompt += "          \"meal_time\": \"19:00\",\n"
	prompt += "          \"meridiem\": \"PM\",\n"
	prompt += "          \"foods\": [\n"
	prompt += "            {\"name\": \"Salmon\", \"portion_ratio\": 40},\n"
	prompt += "            {\"name\": \"Sweet Potato\", \"portion_ratio\": 30},\n"
	prompt += "            {\"name\": \"Spinach\", \"portion_ratio\": 15},\n"
	prompt += "            {\"name\": \"Avocado\", \"portion_ratio\": 15}\n"
	prompt += "          ]\n"
	prompt += "        }\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  },\n"
	prompt += "  \"prepare\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Preparing Protein\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Keep seasoning simple: salt, pepper, garlic powder\",\n"
	prompt += "        \"Batch-cook ground meats: press ~5 lb onto sheet pan, season, bake\",\n"
	prompt += "        \"Slow-cook chicken for 6-8 hours; shred for easy portioning\",\n"
	prompt += "        \"Sheet-pan basics: 8-10 chicken breasts or whole salmon filet on foil\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Preparing Carbs\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Batch cook legumes, oats, pasta, rice, potatoes\",\n"
	prompt += "        \"Use rice cooker for convenience\",\n"
	prompt += "        \"Roast potatoes at 400°F with salt and pepper\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Preparing Fat\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Use whole-food fats: avocado, nuts, seeds, nut butters\",\n"
	prompt += "        \"Protein-with-fat options: salmon, trout, steak\",\n"
	prompt += "        \"NO oils or condiments - whole foods only\"\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  ],\n"
	prompt += "  \"cook\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Cook Protein\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Use 400°F (oven or air fryer) for most proteins\",\n"
	prompt += "        \"Season with salt, pepper, garlic powder\",\n"
	prompt += "        \"Batch options: ground meat sheet-pan (~25 min at 400°F)\",\n"
	prompt += "        \"Steaks: grill about 9 minutes per side for medium-rare\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Cook Carbs\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Pasta boils for ~12 minutes al dente\",\n"
	prompt += "        \"Rice & grains: use 2:1 water-to-grain ratio in rice cooker\",\n"
	prompt += "        \"Oven potatoes: season with salt and pepper; bake at 400°F for ~35-40 minutes\",\n"
	prompt += "        \"Air-fryer potatoes: 400°F for ~15-20 min\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Cook Fat\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Most fats are add-ins: cheese, nuts, nut butters\",\n"
	prompt += "        \"No cooking required for most fat sources\"\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  ],\n"
	prompt += "  \"weight_assemble\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Food Scale Basics\",\n"
	prompt += "      \"subtitle\": \"Why GRAMS (not servings/oz)\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Consistent across foods; servings/ounces vary, grams don't\",\n"
	prompt += "        \"Faster visual learning → you'll 'see' portions and later track less\",\n"
	prompt += "        \"Example plate: 85g chicken, 80g rice, 100g broccoli, 60g avocado\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Food Weight vs. Macro Grams\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Food weight (g) ≠ macro grams\",\n"
	prompt += "        \"Example: 100g chicken breast → ~31g protein, 0g carbs, ~3g fat\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"How to Use a Food Scale\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Put plate on scale\",\n"
	prompt += "        \"Tare (zero it)\",\n"
	prompt += "        \"Add first food → log the grams\",\n"
	prompt += "        \"Tare again\",\n"
	prompt += "        \"Repeat for each food\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Assemble Your Meals\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Think assembly, not recipes: combine building blocks\",\n"
	prompt += "        \"Wrap template: tortilla + black beans + egg whites + cheese + guacamole\",\n"
	prompt += "        \"Bowl template: roasted veg base + rice/potatoes + salmon/chicken + sauce\",\n"
	prompt += "        \"Salad template: lettuce base + beans/potatoes + protein; keep dressing separate\",\n"
	prompt += "        \"Add fats at the end for easier macro control\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Storage\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Short-term (3-4 days): store cooked foods in airtight containers\",\n"
	prompt += "        \"Freeze proteins: cool, break up, flat-freeze in zip bags\",\n"
	prompt += "        \"Portion before storing: weigh into meal-sized servings\",\n"
	prompt += "        \"Label + rotate: write item + date; use oldest first (FIFO)\"\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  ]\n"
	prompt += "}\n\n"

	prompt += "IMPORTANT:\n"
	prompt += "- Return ONLY the JSON object, no additional text\n"
	prompt += "- Generate meals for " + fmt.Sprintf("%d days", len(dates)) + "\n"
	prompt += "- Do NOT change or modify the dates - use them exactly as provided\n"
	prompt += "- Do NOT add extra dates beyond what was requested\n"
	prompt += "- Generate meals ONLY for the specified dates, no more, no less\n"
	prompt += "- Calculate portion ratios to help achieve the per-meal macro targets\n"
	prompt += "- Consider protein content for muscle building, carbs for energy, fats for satiety\n"
	prompt += "- Use realistic, healthy food combinations\n"
	prompt += "- Vary the foods across days to provide variety\n"
	prompt += "- Consider the user's diet type and restrictions\n"
	prompt += "- FOLLOW THE 4-COMPONENT RULE: Every meal must have protein, starchy carb, fruit/vegetable, and fat\n"
	prompt += "- ENFORCE 50/50 CARB SPLIT: Half starchy carbs, half fruits/vegetables\n"
	prompt += "- USE BREAKFAST FOODS ONLY for breakfast meals\n"
	prompt += "- SPECIFY GRAMS AND COOKED/RAW for all portions\n"
	prompt += "- PRIORITIZE WHOLE-FOOD FATS over oils\n"
	prompt += "- DO NOT INCLUDE OILS OR CONDIMENTS: Never add oils (olive oil, vegetable oil, coconut oil, etc.) or condiments (ketchup, mustard, mayonnaise, etc.) to meals\n"
	prompt += "- RESTRICT MULTI-INGREDIENT FOODS: Avoid foods with multiple ingredients (processed foods, packaged items, complex recipes). Use single-ingredient whole foods only\n"
	prompt += "- NO FOOD COUNT RESTRICTION: Use as many food items as needed to fulfill macro targets - there is no limit on the number of foods per meal\n\n"

	prompt += "Create the meal plan now:"

	return prompt
}

func (gs *GeminiService) buildSingleMealPrompt(reqBody models.RequestBody, day string, mealName string, mealTime string, meridiem string, previousMeals []models.MealLLMItems) string {
	prompt := "You are a professional nutritionist and meal planning expert. Create a single high-quality meal based on the user's requirements.\n\n"

	// Parse meals per day
	mealsPerDay := 3
	if reqBody.MealsPerDay != "" {
		if parsed, err := strconv.Atoi(reqBody.MealsPerDay); err == nil {
			mealsPerDay = parsed
		}
	}

	prompt += "USER PROFILE:\n"
	if reqBody.Name != "" {
		prompt += fmt.Sprintf("- Name: %s\n", reqBody.Name)
	}
	if reqBody.Age > 0 {
		prompt += fmt.Sprintf("- Age: %d years\n", reqBody.Age)
	}
	if reqBody.Gender != "" {
		prompt += fmt.Sprintf("- Gender: %s\n", reqBody.Gender)
	}
	if reqBody.Weight > 0 {
		prompt += fmt.Sprintf("- Weight: %d kg\n", reqBody.Weight)
	}
	if reqBody.Height > 0 {
		prompt += fmt.Sprintf("- Height: %d cm\n", reqBody.Height)
	}
	if reqBody.Goal != "" {
		prompt += fmt.Sprintf("- Goal: %s\n", reqBody.Goal)
	}
	if reqBody.ActivityLevel != "" {
		prompt += fmt.Sprintf("- Activity Level: %s\n", reqBody.ActivityLevel)
	}
	prompt += fmt.Sprintf("- Diet Type: %s\n", reqBody.DietType)
	prompt += "\n"

	// Add context from previous meals to avoid repetition
	if len(previousMeals) > 0 {
		prompt += "PREVIOUS MEALS (AVOID REPEATING THESE FOODS):\n"
		usedFoods := make(map[string]bool)
		for _, prevMeal := range previousMeals {
			for _, food := range prevMeal.Foods {
				if !usedFoods[food.Name] {
					prompt += fmt.Sprintf("- %s (used in %s)\n", food.Name, prevMeal.MealName)
					usedFoods[food.Name] = true
				}
			}
		}
		prompt += "\n"
	}

	prompt += "CURRENT MEAL TO GENERATE:\n"
	prompt += fmt.Sprintf("- Day: %s\n", day)
	prompt += fmt.Sprintf("- Meal Name: %s\n", mealName)
	prompt += fmt.Sprintf("- Meal Time: %s %s\n", mealTime, meridiem)
	prompt += "\n"

	prompt += "MACRO TARGETS FOR THIS MEAL:\n"
	prompt += fmt.Sprintf("- Calories: %.1f\n", reqBody.DailyCaloriesGoal/float64(mealsPerDay))
	prompt += fmt.Sprintf("- Protein: %.1fg\n", reqBody.DailyProtiensGoal/float64(mealsPerDay))
	prompt += fmt.Sprintf("- Carbs: %.1fg\n", reqBody.DailyCarbsGoal/float64(mealsPerDay))
	prompt += fmt.Sprintf("- Fats: %.1fg\n", reqBody.DailyFatsGoal/float64(mealsPerDay))
	prompt += "\n"

	if len(reqBody.FoodAllergies) > 0 {
		prompt += fmt.Sprintf("ALLERGIES/FOODS TO AVOID: %s\n", strings.Join(reqBody.FoodAllergies, ", "))
	}
	if len(reqBody.FoodLikes) > 0 {
		prompt += fmt.Sprintf("FOOD PREFERENCES (PRIORITIZE): %s\n", strings.Join(reqBody.FoodLikes, ", "))
	}
	if len(reqBody.SelectedHealthConditions) > 0 {
		prompt += fmt.Sprintf("HEALTH CONDITIONS: %s\n", strings.Join(reqBody.SelectedHealthConditions, ", "))
	}
	if len(reqBody.SelectedLifeStages) > 0 {
		prompt += fmt.Sprintf("LIFE STAGES: %s\n", strings.Join(reqBody.SelectedLifeStages, ", "))
	}
	prompt += "\n"

	prompt += "MEAL GENERATION RULES:\n"
	prompt += "1. UNIVERSAL MEAL STRUCTURE (4-Component Rule):\n"
	prompt += "   - Component 1: Protein Source (chicken, fish, beef, turkey, eggs, Greek yogurt, tofu)\n"
	prompt += "   - Component 2: Starchy Carbohydrate (50% of meal carbs) - rice, oats, potatoes, sweet potatoes, pasta, quinoa, bread, corn\n"
	prompt += "   - Component 3: Fruit or Vegetable (50% of meal carbs) - berries, apples, bananas, broccoli, peppers, spinach, mixed greens, carrots, tomatoes\n"
	prompt += "   - Component 4: Fat Source (whole-food priority: avocado, nuts, seeds, nut butters, cheese)\n\n"

	prompt += "2. MACRO DISTRIBUTION:\n"
	prompt += "   - Daily: 40% Carbs | 30% Protein | 30% Fat (fat target MUST be met)\n"
	prompt += "   - Per-Meal: Divide daily targets by number of meals\n"
	prompt += "   - CRITICAL: Split carbs 50% starchy / 50% fruit-vegetable\n"
	prompt += "   - If fat is under target after protein/carb planning, add a whole-food fat component to reach the fat target.\n\n"

	prompt += "3. HIERARCHICAL PLANNING:\n"
	prompt += "   - STEP 1: Plan carbohydrate sources first (50/50 split)\n"
	prompt += "   - STEP 2: Plan protein sources second\n"
	prompt += "   - STEP 3: Complete with fat source if needed (always include a fat component)\n\n"

	prompt += "4. BREAKFAST FOODS (for breakfast meals only):\n"
	prompt += "   - Eggs, dairy (Greek yogurt, cottage cheese, milk, cheese)\n"
	prompt += "   - Grains: Oats, cereals, granola, whole wheat bread, English muffins\n"
	prompt += "   - Proteins: Turkey bacon, Canadian bacon, breakfast sausage\n"
	prompt += "   - Fruits: Any fruits (berries, bananas, apples, etc.)\n"
	prompt += "   - Other: Avocado, nut butters, nuts, seeds, protein powder\n\n"

	prompt += "5. PORTION SPECIFICATIONS:\n"
	prompt += "   - ALL portions MUST be in GRAMS ONLY (never cups, ounces, tablespoons)\n"
	prompt += "   - Specify (cooked) or (raw) for meats, grains, starchy vegetables\n"
	prompt += "   - Examples: '150g chicken breast (cooked)', '185g brown rice (cooked)', '200g sweet potato (raw)'\n\n"

	prompt += "6. DIETARY RESTRICTIONS:\n"
	prompt += "   - Vegetarian: No meat or fish\n"
	prompt += "   - Vegan: No animal products (meat, fish, dairy, eggs)\n"
	prompt += "   - Pescatarian: Fish only, no other meat\n"
	prompt += "   - Paleo: Whole foods, no grains, dairy, or legumes\n"
	prompt += "   - Gluten-Free: No wheat, barley, rye\n"
	prompt += "   - Dairy-Free: No milk products\n\n"

	prompt += "7. CRITICAL RULES:\n"
	prompt += "   - 50/50 Carb Split: ALWAYS split carbs 50% starchy / 50% fruit-vegetable\n"
	prompt += "   - Whole-Food Fat Priority: Use nuts, seeds, avocado, nut butters BEFORE oils\n"
	prompt += "   - NO OILS OR CONDIMENTS: DO NOT include any oils (olive oil, vegetable oil, coconut oil, etc.) or condiments (ketchup, mustard, mayonnaise, etc.) in meals\n"
	prompt += "   - RESTRICT MULTI-INGREDIENT FOODS: Avoid foods with multiple ingredients (processed foods, packaged items, complex recipes). Use single-ingredient whole foods only\n"
	prompt += "   - NO FOOD COUNT RESTRICTION: Use as many food items as needed to fulfill macro targets - there is no limit on the number of foods per meal\n"
	prompt += "   - Protein/Fat Balancing: If high-fat protein reaches fat limit before protein target, add low-fat protein source\n"
	prompt += "   - Breakfast Foods Enforcement: Breakfast meals = breakfast foods ONLY\n"
	prompt += "   - Grams Only: All portions in grams (never cups, oz, tbsp)\n"
	prompt += "   - Cooked/Raw Required: All meats, grains, starchy veggies MUST specify cooked/raw\n\n"

	prompt += "VARIETY & REALISM:\n"
	prompt += "- Do not repeat any foods from the previous meals list above.\n"
	prompt += "- Avoid repeating the same primary protein for the same meal name on consecutive days.\n"
	prompt += "- Use realistic combinations from different cuisines.\n\n"

	prompt += "RESPONSE FORMAT (JSON ONLY):\n"
	prompt += "{\n"
	prompt += "  \"success\": true,\n"
	prompt += "  \"message\": \"Meal created\",\n"
	prompt += "  \"data\": {\n"
	prompt += fmt.Sprintf("    \"meal_name\": \"%s\",\n", mealName)
	prompt += fmt.Sprintf("    \"meal_time\": \"%s\",\n", mealTime)
	prompt += fmt.Sprintf("    \"meridiem\": \"%s\",\n", meridiem)
	prompt += "    \"foods\": [\n"
	prompt += "      {\"name\": \"Food Name 1\", \"portion_ratio\": 40},\n"
	prompt += "      {\"name\": \"Food Name 2\", \"portion_ratio\": 30},\n"
	prompt += "      {\"name\": \"Food Name 3\", \"portion_ratio\": 20},\n"
	prompt += "      {\"name\": \"Food Name 4\", \"portion_ratio\": 10}\n"
	prompt += "    ]\n"
	prompt += "  }\n"
	prompt += "}\n\n"
	prompt += "IMPORTANT:\n"
	prompt += "- Return ONLY valid JSON, no additional text\n"
	prompt += "- Use as many food items as needed to fulfill macro targets - there is NO restriction on the number of foods per meal\n"
	prompt += "- Ensure macro targets are achievable with the suggested foods\n"
	prompt += "- Prioritize whole, unprocessed foods\n"
	prompt += "- Consider the user's preferences and restrictions\n"

	return prompt
}

func (gs *GeminiService) parseSingleMealResponse(response string, reqBody models.RequestBody, day string, mealName string, mealTime string, meridiem string) (*models.MealLLMItems, error) {
	cleanedResponse := gs.cleanLLMResponse(response)

	var mealResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			MealName string                   `json:"meal_name"`
			MealTime string                   `json:"meal_time"`
			Meridiem string                   `json:"meridiem"`
			Foods    []models.FoodWithPortion `json:"foods"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &mealResponse); err != nil {
		log.Printf("Failed to parse single meal JSON: %v", err)
		// Return default meal
		return &models.MealLLMItems{
			MealName: mealName,
			MealTime: mealTime,
			Meridiem: meridiem,
			Foods:    gs.getDefaultFoodsForMeal(mealName, reqBody.DietType, reqBody.FoodAllergies),
		}, nil
	}

	// Calculate macro targets per meal
	mealsPerDay := 3
	if reqBody.MealsPerDay != "" {
		if parsed, err := strconv.Atoi(reqBody.MealsPerDay); err == nil {
			mealsPerDay = parsed
		}
	}

	// Deduplicate foods from Gemini response (same logic as cleanFoodsArrays)
	unique := make(map[string]bool)
	var deduped []models.FoodWithPortion
	for _, fw := range mealResponse.Data.Foods {
		name := strings.TrimSpace(strings.ToLower(fw.Name))
		if name == "" || unique[name] {
			continue // Skip empty or duplicate foods
		}
		unique[name] = true
		deduped = append(deduped, fw)
	}

	// Pad with defaults if needed to ensure at least 4 unique foods
	if len(deduped) < 4 {
		defaults := gs.getDefaultFoodsForMeal(mealName, reqBody.DietType, reqBody.FoodAllergies)
		for _, df := range defaults {
			if len(deduped) >= 4 {
				break
			}
			lname := strings.TrimSpace(strings.ToLower(df.Name))
			if !unique[lname] {
				unique[lname] = true
				deduped = append(deduped, df)
			}
		}
	}

	meal := models.MealLLMItems{
		MealName: mealName,
		MealTime: mealTime,
		Meridiem: meridiem,
		MacroTarget: models.MacroTarget{
			Calories: reqBody.DailyCaloriesGoal / float64(mealsPerDay),
			Carbs:    reqBody.DailyCarbsGoal / float64(mealsPerDay),
			Proteins: reqBody.DailyProtiensGoal / float64(mealsPerDay),
			Fats:     reqBody.DailyFatsGoal / float64(mealsPerDay),
		},
		Foods: deduped, // Use deduplicated foods
	}

	return &meal, nil
}

func (gs *GeminiService) buildRegenerationPrompt(reqBody models.RegenerationRequest) string {
	prompt := "You are a professional nutritionist and meal planning expert. Regenerate a meal based on the user's requirements while maintaining the exact same macro targets.\n\n"

	prompt += "USER REQUIREMENTS:\n"
	prompt += fmt.Sprintf("- Diet Type: %s\n", reqBody.DietType)
	prompt += fmt.Sprintf("- Meal Style: %s\n", reqBody.MealStyle)

	if len(reqBody.FoodsToAvoid) > 0 {
		prompt += fmt.Sprintf("- Foods to Avoid: %s\n", strings.Join(reqBody.FoodsToAvoid, ", "))
	}

	if len(reqBody.FoodsToLike) > 0 {
		prompt += fmt.Sprintf("- Foods to Like: %s\n", strings.Join(reqBody.FoodsToLike, ", "))
	}

	// Original meal information with explicit macro targets
	prompt += "\nORIGINAL MEAL TO REGENERATE:\n"
	prompt += fmt.Sprintf("- Meal Name: %s\n", reqBody.OriginalMeal.MealName)
	prompt += fmt.Sprintf("- Meal Time: %s %s\n", reqBody.OriginalMeal.MealTime, reqBody.OriginalMeal.Meridiem)
	prompt += fmt.Sprintf("- CRITICAL MACRO TARGETS (MUST MAINTAIN): Calories: %.1f, Protein: %.1fg, Carbs: %.1fg, Fat: %.1fg\n",
		reqBody.OriginalMeal.MacroTarget.Calories, reqBody.OriginalMeal.MacroTarget.Proteins,
		reqBody.OriginalMeal.MacroTarget.Carbs, reqBody.OriginalMeal.MacroTarget.Fats)

	prompt += "- Current Foods:\n"
	for _, food := range reqBody.OriginalMeal.Foods {
		prompt += fmt.Sprintf("  * %s\n", food.FoodName)
	}

	// Regeneration instructions
	if len(reqBody.FoodsToRegenerate) > 0 {
		prompt += "\nREGENERATION REQUEST:\n"
		prompt += fmt.Sprintf("Replace these specific foods: %s\n", strings.Join(reqBody.FoodsToRegenerate, ", "))
		prompt += "Keep the same meal structure and EXACTLY the same macro targets.\n"
		prompt += "Provide alternative foods that maintain similar nutritional profiles.\n"
	} else {
		prompt += "\nREGENERATION REQUEST:\n"
		prompt += "Regenerate the entire meal with different foods while maintaining the EXACT same macro targets.\n"
		prompt += "Use as many foods as needed to fulfill macro targets - there is NO restriction on the number of food items.\n"
		prompt += "Maintain proper nutritional balance with protein, carb, and fat sources.\n"
	}

	prompt += "\nCRITICAL REQUIREMENTS:\n"
	prompt += "1. MACRO TARGETS MUST BE IDENTICAL: Use the exact same macro targets as the original meal\n"
	prompt += "2. MEAL STRUCTURE: Use as many foods as needed to fulfill macro targets - there is NO restriction on the number of food items\n"
	prompt += "3. NUTRITIONAL BALANCE: Ensure protein, carb, and fat sources are well-distributed\n\n"

	prompt += "MEAL GENERATION RULES:\n"
	prompt += "1. UNIVERSAL MEAL STRUCTURE (4-Component Rule):\n"
	prompt += "   - Component 1: Protein Source (chicken, fish, beef, turkey, eggs, Greek yogurt, tofu)\n"
	prompt += "   - Component 2: Starchy Carbohydrate (50% of meal carbs) - rice, oats, potatoes, sweet potatoes, pasta, quinoa, bread, corn\n"
	prompt += "   - Component 3: Fruit or Vegetable (50% of meal carbs) - berries, apples, bananas, broccoli, peppers, spinach, mixed greens, carrots, tomatoes\n"
	prompt += "   - Component 4: Fat Source (whole-food priority: avocado, nuts, seeds, nut butters, cheese)\n\n"

	prompt += "2. MACRO DISTRIBUTION:\n"
	prompt += "   - CRITICAL: Use the EXACT macro targets from the original meal\n"
	prompt += "   - Split carbs 50% starchy / 50% fruit-vegetable\n"
	prompt += "   - Ensure fat target is met with whole-food fats\n\n"

	prompt += "3. BREAKFAST FOODS (for breakfast meals only):\n"
	prompt += "   - Eggs, dairy (Greek yogurt, cottage cheese, milk, cheese)\n"
	prompt += "   - Grains: Oats, cereals, granola, whole wheat bread, English muffins\n"
	prompt += "   - Proteins: Turkey bacon, Canadian bacon, breakfast sausage\n"
	prompt += "   - Fruits: Any fruits (berries, bananas, apples, etc.)\n"
	prompt += "   - Other: Avocado, nut butters, nuts, seeds, protein powder\n\n"

	prompt += "4. PORTION SPECIFICATIONS:\n"
	prompt += "   - ALL portions MUST be in GRAMS ONLY (never cups, ounces, tablespoons)\n"
	prompt += "   - Specify (cooked) or (raw) for meats, grains, starchy vegetables\n"
	prompt += "   - Examples: '150g chicken breast (cooked)', '185g brown rice (cooked)', '200g sweet potato (raw)'\n\n"

	prompt += "5. CRITICAL RESTRICTIONS:\n"
	prompt += "   - NO OILS OR CONDIMENTS: DO NOT include any oils (olive oil, vegetable oil, coconut oil, etc.) or condiments (ketchup, mustard, mayonnaise, etc.) in meals\n"
	prompt += "   - RESTRICT MULTI-INGREDIENT FOODS: Avoid foods with multiple ingredients (processed foods, packaged items, complex recipes). Use single-ingredient whole foods only\n"
	prompt += "   - Use whole-food fats only: avocado, nuts, seeds, nut butters, cheese\n"
	prompt += "   - NO FOOD COUNT RESTRICTION: Use as many food items as needed to fulfill macro targets - there is no limit on the number of foods per meal\n\n"

	prompt += "RESPONSE FORMAT:\n"
	prompt += "Return ONLY a valid JSON object in this exact structure:\n"
	prompt += "{\n"
	prompt += "  \"success\": true,\n"
	prompt += "  \"message\": \"Meal regenerated successfully\",\n"
	prompt += "  \"data\": {\n"
	prompt += fmt.Sprintf("    \"meal_name\": \"%s\",\n", reqBody.OriginalMeal.MealName)
	prompt += fmt.Sprintf("    \"meal_time\": \"%s\",\n", reqBody.OriginalMeal.MealTime)
	prompt += fmt.Sprintf("    \"meridiem\": \"%s\",\n", reqBody.OriginalMeal.Meridiem)
	prompt += "    \"macro_target\": {\n"
	prompt += fmt.Sprintf("      \"calories\": %.1f,\n", reqBody.OriginalMeal.MacroTarget.Calories)
	prompt += fmt.Sprintf("      \"proteins\": %.1f,\n", reqBody.OriginalMeal.MacroTarget.Proteins)
	prompt += fmt.Sprintf("      \"carbs\": %.1f,\n", reqBody.OriginalMeal.MacroTarget.Carbs)
	prompt += fmt.Sprintf("      \"fats\": %.1f\n", reqBody.OriginalMeal.MacroTarget.Fats)
	prompt += "    },\n"
	prompt += "    \"foods\": [\n"
	prompt += "      {\"name\": \"Food Name 1\", \"portion_ratio\": 40},\n"
	prompt += "      {\"name\": \"Food Name 2\", \"portion_ratio\": 30},\n"
	prompt += "      {\"name\": \"Food Name 3\", \"portion_ratio\": 20},\n"
	prompt += "      {\"name\": \"Food Name 4\", \"portion_ratio\": 10}\n"
	prompt += "    ]\n"
	prompt += "  },\n"
	prompt += "  \"prepare\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Preparing Protein\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Keep seasoning simple: salt, pepper, garlic powder\",\n"
	prompt += "        \"Batch-cook ground meats: press ~5 lb onto sheet pan, season, bake\",\n"
	prompt += "        \"Slow-cook chicken for 6-8 hours; shred for easy portioning\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Preparing Carbs\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Batch cook legumes, oats, pasta, rice, potatoes\",\n"
	prompt += "        \"Use rice cooker for convenience\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Preparing Fat\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Use whole-food fats: avocado, nuts, seeds, nut butters\"\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  ],\n"
	prompt += "  \"cook\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Cook Protein\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Use 400°F (oven or air fryer) for most proteins\",\n"
	prompt += "        \"Season with salt, pepper, garlic powder\",\n"
	prompt += "        \"Batch options: ground meat sheet-pan (~25 min at 400°F)\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Cook Carbs\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Pasta boils for ~12 minutes al dente\",\n"
	prompt += "        \"Rice & grains: use 2:1 water-to-grain ratio in rice cooker\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Cook Fat\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Most fats are add-ins: cheese, nuts, nut butters\",\n"
	prompt += "        \"No cooking required for most fat sources\"\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  ],\n"
	prompt += "  \"weight_assemble\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Food Scale Basics\",\n"
	prompt += "      \"subtitle\": \"Why GRAMS (not servings/oz)\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Consistent across foods; servings/ounces vary, grams don't\",\n"
	prompt += "        \"Faster visual learning → you'll 'see' portions and later track less\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"How to Use a Food Scale\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Put plate on scale\",\n"
	prompt += "        \"Tare (zero it)\",\n"
	prompt += "        \"Add first food → log grams\",\n"
	prompt += "        \"Tare again\",\n"
	prompt += "        \"Repeat for each food\"\n"
	prompt += "      ]\n"
	prompt += "    },\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"Assemble Your Meals\",\n"
	prompt += "      \"subtitle\": \"\",\n"
	prompt += "      \"steps\": [\n"
	prompt += "        \"Wrap template: tortilla + protein + carbs + fats + sauce\",\n"
	prompt += "        \"Bowl template: roasted veg base + rice/potatoes + protein + sauce\",\n"
	prompt += "        \"Add fats at the end for easier macro control\"\n"
	prompt += "      ]\n"
	prompt += "    }\n"
	prompt += "  ]\n"
	prompt += "}\n\n"

	prompt += "CRITICAL INSTRUCTIONS:\n"
	prompt += fmt.Sprintf("- meal_name MUST be exactly: \"%s\"\n", reqBody.OriginalMeal.MealName)
	prompt += fmt.Sprintf("- meal_time MUST be exactly: \"%s\"\n", reqBody.OriginalMeal.MealTime)
	prompt += fmt.Sprintf("- meridiem MUST be exactly: \"%s\"\n", reqBody.OriginalMeal.Meridiem)
	prompt += fmt.Sprintf("- macro_target.calories MUST be exactly: %.1f\n", reqBody.OriginalMeal.MacroTarget.Calories)
	prompt += fmt.Sprintf("- macro_target.proteins MUST be exactly: %.1f\n", reqBody.OriginalMeal.MacroTarget.Proteins)
	prompt += fmt.Sprintf("- macro_target.carbs MUST be exactly: %.1f\n", reqBody.OriginalMeal.MacroTarget.Carbs)
	prompt += fmt.Sprintf("- macro_target.fats MUST be exactly: %.1f\n", reqBody.OriginalMeal.MacroTarget.Fats)
	prompt += "- DO NOT change meal_name, meal_time, meridiem, or macro_target values\n"
	prompt += "- ONLY change the foods array with new food choices\n\n"

	prompt += "IMPORTANT:\n"
	prompt += "- Return ONLY the JSON object, no additional text\n"
	prompt += fmt.Sprintf("- Use EXACTLY these macro targets: Calories=%.1f, Protein=%.1fg, Carbs=%.1fg, Fat=%.1fg\n",
		reqBody.OriginalMeal.MacroTarget.Calories, reqBody.OriginalMeal.MacroTarget.Proteins,
		reqBody.OriginalMeal.MacroTarget.Carbs, reqBody.OriginalMeal.MacroTarget.Fats)
	prompt += "- Use as many foods as needed with realistic portion ratios - there is NO restriction on the number of food items\n"
	prompt += "- FOLLOW THE 4-COMPONENT RULE: Every meal must have protein, starchy carb, fruit/vegetable, and fat\n"
	prompt += "- ENFORCE 50/50 CARB SPLIT: Half starchy carbs, half fruits/vegetables\n"
	prompt += "- SPECIFY GRAMS AND COOKED/RAW for all portions\n"
	prompt += "- PRIORITIZE WHOLE-FOOD FATS over oils\n"
	prompt += "- DO NOT INCLUDE OILS OR CONDIMENTS: Never add oils (olive oil, vegetable oil, coconut oil, etc.) or condiments (ketchup, mustard, mayonnaise, etc.) to meals\n"
	prompt += "- RESTRICT MULTI-INGREDIENT FOODS: Avoid foods with multiple ingredients (processed foods, packaged items, complex recipes). Use single-ingredient whole foods only\n\n"

	prompt += "Regenerate the meal now:"

	return prompt
}

func (gs *GeminiService) parseMealResponse(response string, reqBody models.RequestBody) (*models.MealPlanLLMResponse, error) {
	// Clean the response first
	cleanedResponse := gs.cleanLLMResponse(response)

	// Try to parse as JSON
	var mealPlan models.MealPlanLLMResponse
	if err := json.Unmarshal([]byte(cleanedResponse), &mealPlan); err != nil {
		log.Printf("Failed to parse JSON response: %v", err)
		return gs.createStructuredResponse(cleanedResponse, reqBody), nil
	}

	// Clean and validate the parsed response
	mealPlan = gs.cleanFoodsArrays(mealPlan, reqBody)
	mealPlan = gs.setMacroTargets(mealPlan, reqBody)

	return &mealPlan, nil
}

func (gs *GeminiService) parseRegenerationResponse(response string, reqBody models.RegenerationRequest) (*models.RegenerationLLMResponse, error) {
	// Clean the response first
	cleanedResponse := gs.cleanLLMResponse(response)

	// Try to parse as JSON
	var regenResponse models.RegenerationLLMResponse
	if err := json.Unmarshal([]byte(cleanedResponse), &regenResponse); err != nil {
		log.Printf("Failed to parse regeneration JSON response: %v", err)
		return gs.createRegenerationStructuredResponse(cleanedResponse, reqBody), nil
	}

	// Validate and fix macro targets if needed
	regenResponse = gs.validateAndFixRegenerationMacros(regenResponse, reqBody)

	// Clean and validate the parsed response
	regenResponse = gs.cleanRegenerationFoods(regenResponse, reqBody)

	return &regenResponse, nil
}

func (gs *GeminiService) createStructuredResponse(response string, reqBody models.RequestBody) *models.MealPlanLLMResponse {
	// Create a structured response with default meals
	mealPlan := models.MealPlanLLMResponse{
		Success: true,
		Message: "Meal plan created successfully",
		Data:    make(map[string]models.DayLLMMeals),
		Prepare: []models.PrepareCookSection{
			{
				Title:    "Preparing Protein",
				Subtitle: "",
				Steps: []string{
					"Keep seasoning simple: salt, pepper, garlic powder",
					"Batch-cook ground meats: press ~5 lb onto sheet pan, season, bake",
					"Slow-cook chicken for 6-8 hours; shred for easy portioning",
				},
			},
			{
				Title:    "Preparing Carbs",
				Subtitle: "",
				Steps: []string{
					"Batch cook legumes, oats, pasta, rice, potatoes",
					"Use rice cooker for convenience",
					"Roast potatoes at 400°F with salt and pepper",
				},
			},
			{
				Title:    "Preparing Fat",
				Subtitle: "",
				Steps: []string{
					"Use whole-food fats: avocado, nuts, seeds, nut butters",
					"Protein-with-fat options: salmon, trout, steak",
					"NO oils or condiments - whole foods only",
				},
			},
		},
		Cook: []models.PrepareCookSection{
			{
				Title:    "Cook Protein",
				Subtitle: "",
				Steps: []string{
					"Use 400°F (oven or air fryer) for most proteins",
					"Season with salt, pepper, garlic powder",
					"Batch options: ground meat sheet-pan (~25 min at 400°F)",
				},
			},
			{
				Title:    "Cook Carbs",
				Subtitle: "",
				Steps: []string{
					"Pasta boils for ~12 minutes al dente",
					"Rice & grains: use 2:1 water-to-grain ratio in rice cooker",
					"Oven potatoes: season with salt and pepper; bake at 400°F for ~35-40 minutes",
				},
			},
			{
				Title:    "Cook Fat",
				Subtitle: "",
				Steps: []string{
					"Most fats are add-ins: cheese, nuts, nut butters",
					"No cooking required for most fat sources",
				},
			},
		},
		WeightAssemble: []models.WeightAssembleSection{
			{
				Title:    "Food Scale Basics",
				Subtitle: "Why GRAMS (not servings/oz)",
				Steps: []string{
					"Consistent across foods; servings/ounces vary, grams don't",
					"Faster visual learning → you'll 'see' portions and later track less",
					"Example plate: 85g chicken, 80g rice, 100g broccoli, 60g avocado",
				},
			},
			{
				Title:    "Food Weight vs. Macro Grams",
				Subtitle: "",
				Steps: []string{
					"Food weight (g) ≠ macro grams",
					"Example: 100g chicken breast → ~31g protein, 0g carbs, ~3g fat",
				},
			},
			{
				Title:    "How to Use a Food Scale",
				Subtitle: "",
				Steps: []string{
					"Put plate on scale",
					"Tare (zero it)",
					"Add first food → log the grams",
					"Tare again",
					"Repeat for each food",
				},
			},
			{
				Title:    "Assemble Your Meals",
				Subtitle: "",
				Steps: []string{
					"Think assembly, not recipes: combine building blocks",
					"Wrap template: tortilla + black beans + egg whites + cheese + guacamole",
					"Bowl template: roasted veg base + rice/potatoes + salmon/chicken + sauce",
					"Add fats at the end for easier macro control",
				},
			},
			{
				Title:    "Storage",
				Subtitle: "",
				Steps: []string{
					"Short-term (3-4 days): store cooked foods in airtight containers",
					"Freeze proteins: cool, break up, flat-freeze in zip bags",
					"Portion before storing: weigh into meal-sized servings",
					"Label + rotate: write item + date; use oldest first (FIFO)",
				},
			},
		},
	}

	// Use dates from request, or default to 7 days if not provided
	dates := reqBody.Dates
	if len(dates) == 0 {
		// Generate 7 days starting from today if no dates provided
		for i := 0; i < 7; i++ {
			dates = append(dates, fmt.Sprintf("2024-01-%02d", i+1))
		}
	}

	meals := []string{"Breakfast", "Lunch", "Dinner"}

	for _, dateKey := range dates {
		dayMeals := models.DayLLMMeals{
			Date:  dateKey,
			Meals: make([]models.MealLLMItems, 3),
		}

		for j, mealName := range meals {
			// Get default foods for this meal
			defaultFoods := gs.getDefaultFoodsForMeal(mealName, reqBody.DietType, reqBody.FoodAllergies)

			dayMeals.Meals[j] = models.MealLLMItems{
				MealName: mealName,
				MealTime: fmt.Sprintf("%02d:00", 8+j*5), // 8 AM, 1 PM, 6 PM
				Meridiem: "AM",
				Foods:    defaultFoods,
			}

			if j > 0 {
				dayMeals.Meals[j].Meridiem = "PM"
			}
		}

		mealPlan.Data[dateKey] = dayMeals
	}

	return &mealPlan
}

func (gs *GeminiService) createRegenerationStructuredResponse(response string, reqBody models.RegenerationRequest) *models.RegenerationLLMResponse {
	// Create a structured response with the regenerated meal using original meal data
	regenResponse := models.RegenerationLLMResponse{
		Success: true,
		Message: "Meal regenerated successfully",
		Prepare: []models.PrepareCookSection{
			{
				Title:    "Preparing Protein",
				Subtitle: "",
				Steps: []string{
					"Keep seasoning simple: salt, pepper, garlic powder",
					"Batch-cook ground meats: press ~5 lb onto sheet pan, season, bake",
					"Slow-cook chicken for 6-8 hours; shred for easy portioning",
				},
			},
			{
				Title:    "Preparing Carbs",
				Subtitle: "",
				Steps: []string{
					"Batch cook legumes, oats, pasta, rice, potatoes",
					"Use rice cooker for convenience",
				},
			},
			{
				Title:    "Preparing Fat",
				Subtitle: "",
				Steps: []string{
					"Use whole-food fats: avocado, nuts, seeds, nut butters",
				},
			},
		},
		Cook: []models.PrepareCookSection{
			{
				Title:    "Cook Protein",
				Subtitle: "",
				Steps: []string{
					"Use 400°F (oven or air fryer) for most proteins",
					"Season with salt, pepper, garlic powder",
					"Batch options: ground meat sheet-pan (~25 min at 400°F)",
				},
			},
			{
				Title:    "Cook Carbs",
				Subtitle: "",
				Steps: []string{
					"Pasta boils for ~12 minutes al dente",
					"Rice & grains: use 2:1 water-to-grain ratio in rice cooker",
				},
			},
			{
				Title:    "Cook Fat",
				Subtitle: "",
				Steps: []string{
					"Most fats are add-ins: cheese, nuts, nut butters",
					"No cooking required for most fat sources",
				},
			},
		},
		WeightAssemble: []models.WeightAssembleSection{
			{
				Title:    "Food Scale Basics",
				Subtitle: "Why GRAMS (not servings/oz)",
				Steps: []string{
					"Consistent across foods; servings/ounces vary, grams don't",
					"Faster visual learning → you'll 'see' portions and later track less",
				},
			},
			{
				Title:    "How to Use a Food Scale",
				Subtitle: "",
				Steps: []string{
					"Put plate on scale",
					"Tare (zero it)",
					"Add first food → log grams",
					"Tare again",
					"Repeat for each food",
				},
			},
			{
				Title:    "Assemble Your Meals",
				Subtitle: "",
				Steps: []string{
					"Wrap template: tortilla + protein + carbs + fats + sauce",
					"Bowl template: roasted veg base + rice/potatoes + protein + sauce",
					"Add fats at the end for easier macro control",
				},
			},
		},
		Data: models.RegenerationLLMData{
			MealName:    reqBody.OriginalMeal.MealName,
			MealTime:    reqBody.OriginalMeal.MealTime,
			Meridiem:    reqBody.OriginalMeal.Meridiem,
			MacroTarget: reqBody.OriginalMeal.MacroTarget, // Use original macro targets
			Foods:       gs.getDefaultFoodsForMeal(reqBody.OriginalMeal.MealName, reqBody.DietType, reqBody.FoodsToAvoid),
		},
	}

	return &regenResponse
}

func (gs *GeminiService) validateAndFixRegenerationMacros(regenResponse models.RegenerationLLMResponse, reqBody models.RegenerationRequest) models.RegenerationLLMResponse {
	// Always preserve original meal identity and macro targets
	regenResponse.Data.MealName = reqBody.OriginalMeal.MealName
	regenResponse.Data.MealTime = reqBody.OriginalMeal.MealTime
	regenResponse.Data.Meridiem = reqBody.OriginalMeal.Meridiem
	regenResponse.Data.MacroTarget = reqBody.OriginalMeal.MacroTarget

	// Log if we had to fix anything
	if regenResponse.Data.MacroTarget.Calories == 0 || regenResponse.Data.MacroTarget.Proteins == 0 {
		log.Printf("Regeneration: Fixed missing macro targets, using original meal targets")
	}
	if regenResponse.Data.MealName != reqBody.OriginalMeal.MealName {
		log.Printf("Regeneration: Fixed meal name from '%s' to '%s'", regenResponse.Data.MealName, reqBody.OriginalMeal.MealName)
	}
	if regenResponse.Data.MealTime != reqBody.OriginalMeal.MealTime {
		log.Printf("Regeneration: Fixed meal time from '%s' to '%s'", regenResponse.Data.MealTime, reqBody.OriginalMeal.MealTime)
	}

	return regenResponse
}

func (gs *GeminiService) cleanRegenerationFoods(regenResponse models.RegenerationLLMResponse, reqBody models.RegenerationRequest) models.RegenerationLLMResponse {
	// Ensure at least 4 unique foods per meal by padding from defaults
	unique := make(map[string]bool)
	var deduped []models.FoodWithPortion
	for _, fw := range regenResponse.Data.Foods {
		name := strings.TrimSpace(strings.ToLower(fw.Name))
		if name == "" || unique[name] {
			continue
		}
		unique[name] = true
		deduped = append(deduped, fw)
	}

	// Pad with defaults if needed
	if len(deduped) < 4 {
		defaults := gs.getDefaultFoodsForMeal(regenResponse.Data.MealName, reqBody.DietType, reqBody.FoodsToAvoid)
		for _, df := range defaults {
			if len(deduped) >= 4 {
				break
			}
			lname := strings.TrimSpace(strings.ToLower(df.Name))
			if !unique[lname] {
				unique[lname] = true
				deduped = append(deduped, df)
			}
		}
	}

	regenResponse.Data.Foods = deduped
	return regenResponse
}

func (gs *GeminiService) extractFoodsFromText(text string) []string {
	// Simple extraction of food names from text
	words := strings.Fields(text)
	var foods []string

	for _, word := range words {
		// Clean the word
		cleanWord := strings.Trim(word, ".,!?;:\"'()[]{}")
		cleanWord = strings.ToLower(cleanWord)

		// Skip common words and short words
		if len(cleanWord) < 3 || gs.isCommonWord(cleanWord) {
			continue
		}

		// Check if it looks like a food name
		if gs.looksLikeFood(cleanWord) {
			foods = append(foods, strings.Title(cleanWord))
		}
	}

	return gs.removeDuplicates(foods)
}

func (gs *GeminiService) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true, "on": true, "at": true,
		"to": true, "for": true, "of": true, "with": true, "by": true, "from": true, "up": true,
		"about": true, "into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "among": true, "under": true, "over": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "you": true, "he": true,
		"she": true, "it": true, "we": true, "they": true, "me": true, "him": true, "her": true,
		"us": true, "them": true, "my": true, "your": true, "his": true, "its": true,
		"our": true, "their": true, "is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true, "may": true,
		"might": true, "must": true, "can": true, "shall": true, "a": true, "an": true,
	}
	return commonWords[word]
}

func (gs *GeminiService) looksLikeFood(word string) bool {
	// Simple heuristic to identify potential food names
	foodIndicators := []string{"chicken", "beef", "fish", "salmon", "rice", "pasta", "bread", "egg", "milk", "cheese", "apple", "banana", "orange", "vegetable", "fruit", "meat", "grain", "nut", "seed", "oil", "butter", "yogurt", "cereal", "oatmeal", "quinoa", "lentil", "bean", "tomato", "potato", "onion", "garlic", "spinach", "lettuce", "carrot", "broccoli", "cauliflower", "cabbage", "pepper", "cucumber", "avocado", "lemon", "lime", "grape", "strawberry", "blueberry", "raspberry", "blackberry", "peach", "pear", "plum", "cherry", "grapefruit", "pineapple", "mango", "kiwi", "papaya", "coconut", "almond", "walnut", "pecan", "cashew", "pistachio", "hazelnut", "macadamia", "brazil", "sunflower", "pumpkin", "sesame", "flax", "chia", "hemp", "olive", "coconut", "canola", "vegetable", "corn", "soybean", "safflower", "grapeseed", "avocado", "walnut", "almond", "peanut", "sesame", "sunflower", "pumpkin", "flax", "chia", "hemp", "olive", "coconut", "canola", "vegetable", "corn", "soybean", "safflower", "grapeseed"}

	for _, indicator := range foodIndicators {
		if strings.Contains(word, indicator) {
			return true
		}
	}
	return false
}

func (gs *GeminiService) removeDuplicates(foods []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, food := range foods {
		if !seen[food] {
			seen[food] = true
			result = append(result, food)
		}
	}

	return result
}

func (gs *GeminiService) getDefaultFoodsForMeal(mealName, dietType string, foodsToAvoid []string) []models.FoodWithPortion {
	// Default food suggestions with portion ratios based on meal and diet type
	defaultFoods := map[string][]models.FoodWithPortion{
		"Breakfast": {
			{Name: "Oatmeal", PortionRatio: 40},
			{Name: "Greek Yogurt", PortionRatio: 25},
			{Name: "Banana", PortionRatio: 20},
			{Name: "Almonds", PortionRatio: 15},
		},
		"Lunch": {
			{Name: "Grilled Chicken Breast", PortionRatio: 40},
			{Name: "Brown Rice", PortionRatio: 30},
			{Name: "Broccoli", PortionRatio: 15},
			{Name: "Avocado", PortionRatio: 15},
		},
		"Dinner": {
			{Name: "Salmon", PortionRatio: 40},
			{Name: "Sweet Potato", PortionRatio: 30},
			{Name: "Spinach", PortionRatio: 15},
			{Name: "Olive Oil", PortionRatio: 15},
		},
	}

	if foods, exists := defaultFoods[mealName]; exists {
		return foods
	}

	return []models.FoodWithPortion{
		{Name: "Chicken Breast", PortionRatio: 40},
		{Name: "Brown Rice", PortionRatio: 30},
		{Name: "Broccoli", PortionRatio: 15},
		{Name: "Avocado", PortionRatio: 15},
	}
}

func (gs *GeminiService) cleanLLMResponse(response string) string {
	// Remove any markdown formatting
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")

	// Remove any leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Try to find JSON object boundaries
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start != -1 && end != -1 && end > start {
		response = response[start : end+1]
	}

	return response
}

func (gs *GeminiService) cleanFoodsArrays(mealPlan models.MealPlanLLMResponse, reqBody models.RequestBody) models.MealPlanLLMResponse {
	// Clean up any empty or invalid food arrays
	for dayKey, dayMeals := range mealPlan.Data {
		for i, meal := range dayMeals.Meals {
			// If empty, seed with defaults
			if len(meal.Foods) == 0 {
				dayMeals.Meals[i].Foods = gs.getDefaultFoodsForMeal(meal.MealName, reqBody.DietType, reqBody.FoodAllergies)
			}

			// Ensure at least 4 unique foods per meal by padding from defaults
			unique := make(map[string]bool)
			var deduped []models.FoodWithPortion
			for _, fw := range dayMeals.Meals[i].Foods {
				name := strings.TrimSpace(strings.ToLower(fw.Name))
				if name == "" || unique[name] {
					continue
				}
				unique[name] = true
				deduped = append(deduped, fw)
			}

			// Pad with defaults if needed
			if len(deduped) < 4 {
				defaults := gs.getDefaultFoodsForMeal(meal.MealName, reqBody.DietType, reqBody.FoodAllergies)
				for _, df := range defaults {
					if len(deduped) >= 4 {
						break
					}
					lname := strings.TrimSpace(strings.ToLower(df.Name))
					if !unique[lname] {
						unique[lname] = true
						deduped = append(deduped, df)
					}
				}
			}

			dayMeals.Meals[i].Foods = deduped
		}
		mealPlan.Data[dayKey] = dayMeals
	}
	return mealPlan
}

func (gs *GeminiService) setMacroTargets(mealPlan models.MealPlanLLMResponse, reqBody models.RequestBody) models.MealPlanLLMResponse {
	// Calculate macro targets per meal based on daily targets and number of meals
	numberOfMeals := reqBody.NumberOfMeals
	if numberOfMeals == 0 && reqBody.MealsPerDay != "" {
		if parsed, err := strconv.Atoi(reqBody.MealsPerDay); err == nil {
			numberOfMeals = parsed
		}
	}
	if numberOfMeals == 0 {
		numberOfMeals = 3 // Default to 3 meals if not specified
	}

	// Calculate per-meal targets by dividing daily targets by number of meals
	caloriesPerMeal := reqBody.DailyCaloriesGoal / float64(numberOfMeals)
	carbsPerMeal := reqBody.DailyCarbsGoal / float64(numberOfMeals)
	proteinsPerMeal := reqBody.DailyProtiensGoal / float64(numberOfMeals)
	fatsPerMeal := reqBody.DailyFatsGoal / float64(numberOfMeals)

	for dayKey, dayMeals := range mealPlan.Data {
		for i := range dayMeals.Meals {
			// Set macro targets based on daily targets divided by number of meals
			dayMeals.Meals[i].MacroTarget = models.MacroTarget{
				Calories: caloriesPerMeal,
				Carbs:    carbsPerMeal,
				Proteins: proteinsPerMeal,
				Fats:     fatsPerMeal,
			}
		}
		mealPlan.Data[dayKey] = dayMeals
	}
	return mealPlan
}

func (gs *GeminiService) prompt(prompt string) (string, error) {
	requestBody := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{
						Text: prompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s?key=%s", gs.baseURL, gs.apiKey)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := gs.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	var response GeminiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	return response.Candidates[0].Content.Parts[0].Text, nil
}
