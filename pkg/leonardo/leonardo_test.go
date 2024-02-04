package leonardo

import (
	"encoding/json"
	"testing"
)

func TestFeedResponse(t *testing.T) {
	data := `{
	"data": {
		"generations": [
			{
				"alchemy": null,
				"contrastRatio": null,
				"highResolution": null,
				"guidanceScale": null,
				"inferenceSteps": null,
				"modelId": null,
				"scheduler": null,
				"coreModel": "SD",
				"sdVersion": null,
				"prompt": "",
				"negativePrompt": null,
				"id": "10000000-0000-0000-0000-000000000000",
				"status": "COMPLETE",
				"quantity": 1,
				"createdAt": "2020-01-01T00:00:00.000",
				"imageHeight": 576,
				"imageWidth": 1024,
				"presetStyle": null,
				"public": false,
				"seed": 6000000000000000,
				"tiling": null,
				"initStrength": null,
				"imageToImage": true,
				"highContrast": false,
				"promptMagic": false,
				"promptMagicVersion": null,
				"promptMagicStrength": null,
				"imagePromptStrength": null,
				"expandedDomain": null,
				"motion": true,
				"photoReal": null,
				"photoRealStrength": null,
				"nsfw": false,
				"user": {
					"username": "username",
					"id": "20000000-0000-0000-0000-000000000000",
					"__typename": "users"
				},
				"custom_model": null,
				"init_image": {
					"id": "30000000-0000-0000-0000-000000000000",
					"url": "https://cdn.leonardo.ai/users/20000000-0000-0000-0000-000000000000/initImages/30000000-0000-0000-0000-000000000000.png",
					"__typename": "init_images"
				},
				"generated_images": [
					{
						"id": "40000000-0000-0000-0000-000000000000",
						"url": "https://cdn.leonardo.ai/users/20000000-0000-0000-0000-000000000000/generations/40000000-0000-0000-0000-000000000000/40000000-0000-0000-0000-000000000000.jpg",
						"motionGIFURL": null,
						"motionMP4URL": "https://cdn.leonardo.ai/users/20000000-0000-0000-0000-000000000000/generations/40000000-0000-0000-0000-000000000000/40000000-0000-0000-0000-000000000000.mp4",
						"likeCount": 0,
						"nsfw": false,
						"generated_image_variation_generics": [],
						"__typename": "generated_images"
					}
				],
				"generation_elements": [],
				"generation_controlnets": [],
				"__typename": "generations"
			}
		]
	}
}`
	var response feedResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		t.Fatal(err)
	}
}

func TestUserResponse(t *testing.T) {
	data := `{
	"data": {
		"users": [
			{
				"id": "10000000-0000-0000-0000-000000000000",
				"username": "username",
				"blocked": false,
				"user_details": [
					{
						"auth0Email": "email@email.email",
						"plan": "BASIC",
						"paidTokens": 0,
						"apiCredit": 100000,
						"subscriptionTokens": 8000,
						"subscriptionModelTokens": 10,
						"subscriptionGptTokens": 1000,
						"subscriptionSource": "STRIPE",
						"interests": [
							"STOCK_IMAGES",
							"BOARD_GAMES",
							"VIDEO_GAMES"
						],
						"interestsRoles": "DEVELOPER",
						"interestsRolesOther": "",
						"showNsfw": true,
						"tokenRenewalDate": "2020-01-01T00:00:00",
						"planSubscribeFrequency": "MONTHLY",
						"apiSubscriptionTokens": null,
						"apiPaidTokens": null,
						"apiPlan": null,
						"paddleId": null,
						"apiPlanAutoTopUpTriggerBalance": null,
						"apiPlanSubscribeFrequency": null,
						"apiPlanSubscribeDate": null,
						"apiPlanSubscriptionSource": null,
						"apiPlanTokenRenewalDate": null,
						"apiPlanTopUpAmount": null,
						"apiConcurrencySlots": 5,
						"__typename": "user_details"
					}
				],
				"team_memberships": [],
				"__typename": "users"
			}
		]
	}
}`
	var response userResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		t.Fatal(err)
	}
}
