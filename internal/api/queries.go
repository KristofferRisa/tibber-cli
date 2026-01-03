package api

// GraphQL queries for Tibber API

// QueryHomes fetches all homes with their details
const QueryHomes = `{
  viewer {
    homes {
      id
      appNickname
      size
      type
      numberOfResidents
      primaryHeatingSource
      hasVentilationSystem
      mainFuseSize
      features {
        realTimeConsumptionEnabled
      }
      address {
        address1
        address2
        address3
        postalCode
        city
        country
        latitude
        longitude
      }
    }
  }
}`

// QueryPrices fetches current and upcoming prices
const QueryPrices = `{
  viewer {
    homes {
      id
      currentSubscription {
        priceInfo {
          current {
            total
            energy
            tax
            startsAt
            level
            currency
          }
          today {
            total
            energy
            tax
            startsAt
            level
            currency
          }
          tomorrow {
            total
            energy
            tax
            startsAt
            level
            currency
          }
        }
      }
    }
  }
}`

// SubscriptionLiveMeasurement is the GraphQL subscription for real-time data
const SubscriptionLiveMeasurement = `subscription($homeId: ID!) {
  liveMeasurement(homeId: $homeId) {
    timestamp
    power
    powerProduction
    accumulatedConsumption
    accumulatedProduction
    accumulatedCost
    accumulatedReward
    minPower
    maxPower
    averagePower
    voltagePhase1
    voltagePhase2
    voltagePhase3
    currentL1
    currentL2
    currentL3
    currency
  }
}`
