package leonardo

var uploadQuery = `mutation CreateUploadInitImage($arg1: InitImageUploadInput!) {
  uploadInitImage(arg1: $arg1) {
    id
    fields
    key
    url
    __typename
  }
}`

var createQuery = `mutation CreateMotionSvdGenerationJob($arg1: MotionSvdGenerationInput!) {
  motionSvdGenerationJob(arg1: $arg1) {
    apiCreditCost
    generationId
    __typename
  }
}`

var statusQuery = `query GetAIGenerationFeedStatuses($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
    __typename
  }
}`

var feedQuery = `query GetAIGenerationFeed($where: generations_bool_exp = {}, $userId: uuid, $limit: Int, $offset: Int = 0) {
  generations(
    limit: $limit
    offset: $offset
    order_by: [{createdAt: desc}]
    where: $where
  ) {
    alchemy
    contrastRatio
    highResolution
    guidanceScale
    inferenceSteps
    modelId
    scheduler
    coreModel
    sdVersion
    prompt
    negativePrompt
    id
    status
    quantity
    createdAt
    imageHeight
    imageWidth
    presetStyle
    sdVersion
    public
    seed
    tiling
    initStrength
    imageToImage
    highContrast
    promptMagic
    promptMagicVersion
    promptMagicStrength
    imagePromptStrength
    expandedDomain
    motion
    photoReal
    photoRealStrength
    nsfw
    user {
      username
      id
      __typename
    }
    custom_model {
      id
      userId
      name
      modelHeight
      modelWidth
      __typename
    }
    init_image {
      id
      url
      __typename
    }
    generated_images(order_by: [{url: desc}]) {
      id
      url
      motionGIFURL
      motionMP4URL
      likeCount
      nsfw
      generated_image_variation_generics(order_by: [{createdAt: desc}]) {
        url
        status
        createdAt
        id
        transformType
        upscale_details {
          alchemyRefinerCreative
          alchemyRefinerStrength
          oneClicktype
          isOneClick
          id
          variationId
          upscaleMultiplier
          width
          height
          __typename
        }
        __typename
      }
      __typename
    }
    generation_elements {
      id
      lora {
        akUUID
        name
        description
        urlImage
        baseModel
        weightDefault
        weightMin
        weightMax
        __typename
      }
      weightApplied
      __typename
    }
    generation_controlnets(order_by: {controlnetOrder: asc}) {
      id
      weightApplied
      controlnet_definition {
        akUUID
        displayName
        displayDescription
        controlnetType
        __typename
      }
      controlnet_preprocessor_matrix {
        id
        preprocessorName
        __typename
      }
      __typename
    }
    __typename
  }
}`

var userQuery = `query GetUserDetails($userSub: String) {
  users(where: {user_details: {cognitoId: {_eq: $userSub}}}) {
    id
    username
    blocked
    user_details {
      auth0Email
      plan
      paidTokens
      apiCredit
      subscriptionTokens
      subscriptionModelTokens
      subscriptionGptTokens
      subscriptionSource
      interests
      interestsRoles
      interestsRolesOther
      showNsfw
      tokenRenewalDate
      planSubscribeFrequency
      apiSubscriptionTokens
      apiPaidTokens
      apiPlan
      paddleId
      apiPlanAutoTopUpTriggerBalance
      apiPlanSubscribeFrequency
      apiPlanSubscribeDate
      apiPlanSubscriptionSource
      apiPlanTokenRenewalDate
      apiPlanTopUpAmount
      apiConcurrencySlots
      __typename
    }
    team_memberships {
      team {
        akUUID
        id
        modifiedAt
        paidTokens
        paymentPlatformId
        plan
        planCustomTokenRenewalAmount
        planSeats
        planSubscribeDate
        planSubscribeFrequency
        planSubscriptionSource
        planTokenRenewalDate
        subscriptionTokens
        teamLogoUrl
        teamName
        __typename
  	  }
      __typename
    }
    __typename
  }
}`
