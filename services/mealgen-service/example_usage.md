# Serving Selection API Usage

## Overview

The serving selection API allows you to send meal preferences with foods and all available serving options to an LLM, which will select the optimal serving sizes to match target macros.

## Endpoint

`POST /serving-selection`

## Request Format

```json
{
  "meals": [
    {
      "meal_name": "Breakfast",
      "meal_time": "8:00",
      "meridiem": "AM",
      "macro_target": {
        "calories": 500,
        "carbs": 60,
        "proteins": 25,
        "fats": 20
      },
      "foods": [
        {
          "food_id": "12345",
          "food_name": "Oatmeal",
          "food_type": "grain",
          "brand_name": "Quaker",
          "servings": [
            {
              "serving_id": "1",
              "serving_description": "1 cup",
              "measurement_description": "1 cup (81g)",
              "metric_serving_amount": "81",
              "metric_serving_unit": "g",
              "number_of_units": "1",
              "calories": "307",
              "protein": "10.7",
              "carbohydrate": "54.8",
              "fat": "5.3",
              "sugar": "1.1",
              "fiber": "8.2",
              "saturated_fat": "0.9",
              "monounsaturated_fat": "1.6",
              "polyunsaturated_fat": "1.6",
              "cholesterol": "0",
              "sodium": "4",
              "potassium": "293",
              "calcium": "20",
              "iron": "3.4",
              "vitamin_a": "0",
              "vitamin_b": "0.2",
              "vitamin_c": "0",
              "vitamin_d": "0"
            },
            {
              "serving_id": "2",
              "serving_description": "1/2 cup",
              "measurement_description": "1/2 cup (40.5g)",
              "metric_serving_amount": "40.5",
              "metric_serving_unit": "g",
              "number_of_units": "1",
              "calories": "154",
              "protein": "5.4",
              "carbohydrate": "27.4",
              "fat": "2.7",
              "sugar": "0.6",
              "fiber": "4.1",
              "saturated_fat": "0.5",
              "monounsaturated_fat": "0.8",
              "polyunsaturated_fat": "0.8",
              "cholesterol": "0",
              "sodium": "2",
              "potassium": "147",
              "calcium": "10",
              "iron": "1.7",
              "vitamin_a": "0",
              "vitamin_b": "0.1",
              "vitamin_c": "0",
              "vitamin_d": "0"
            }
          ],
          "selected_serving": {
            "serving_id": "",
            "serving_description": "",
            "measurement_description": "",
            "metric_serving_amount": "",
            "metric_serving_unit": "",
            "number_of_units": "",
            "calories": "",
            "protein": "",
            "carbohydrate": "",
            "fat": "",
            "sugar": "",
            "fiber": "",
            "saturated_fat": "",
            "monounsaturated_fat": "",
            "polyunsaturated_fat": "",
            "cholesterol": "",
            "sodium": "",
            "potassium": "",
            "calcium": "",
            "iron": "",
            "vitamin_a": "",
            "vitamin_b": "",
            "vitamin_c": "",
            "vitamin_d": ""
          }
        }
      ]
    }
  ]
}
```

## Response Format

```json
{
  "success": true,
  "data": {
    "Breakfast": {
      "meal_name": "Breakfast",
      "meal_time": "8:00",
      "meridiem": "AM",
      "foods": [
        {
          "food_id": "12345",
          "food_name": "Oatmeal",
          "selected_serving": {
            "serving_id": "1",
            "serving_description": "1 cup",
            "measurement_description": "1 cup (81g)",
            "metric_serving_amount": "81",
            "metric_serving_unit": "g",
            "number_of_units": "1.2",
            "calories": "368.4",
            "protein": "12.8",
            "carbohydrate": "65.8",
            "fat": "6.4",
            "sugar": "1.3",
            "fiber": "9.8",
            "saturated_fat": "1.1",
            "monounsaturated_fat": "1.9",
            "polyunsaturated_fat": "1.9",
            "cholesterol": "0",
            "sodium": "4.8",
            "potassium": "351.6",
            "calcium": "24",
            "iron": "4.1",
            "vitamin_a": "0",
            "vitamin_b": "0.2",
            "vitamin_c": "0",
            "vitamin_d": "0"
          }
        }
      ]
    }
  },
  "message": "Serving selection completed successfully"
}
```

## Key Features

1. **LLM Selection**: The LLM analyzes all available serving options and selects the best one for each food
2. **Macro Optimization**: The LLM calculates optimal `number_of_units` to match target macros
3. **Realistic Portions**: Considers meal type and time for appropriate serving sizes
4. **Fallback Handling**: If LLM parsing fails, defaults to first serving with unit value of 1
5. **Complete Data**: All macro and micronutrient values are recalculated based on selected serving

## How It Works

1. **Input**: Send meal preferences with foods containing all available serving options
2. **LLM Processing**: The LLM analyzes each food's serving options and selects the best one
3. **Macro Optimization**: LLM calculates optimal `number_of_units` to match target macros
4. **Output**: Returns foods with `selected_serving` containing the optimal serving size

## Error Handling

- If LLM parsing fails, the system falls back to selecting the first serving
- All required fields are populated with default values if missing
- The system ensures the API continues to work even with API issues

## Testing

Use the provided `test_serving_selection.json` file to test the functionality:

```bash
curl -X POST http://localhost:8080/serving-selection \
  -H "Content-Type: application/json" \
  -d @test_serving_selection.json
```
