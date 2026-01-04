# Reconciler Blueprint Examples

This document provides example Blueprints for various mining operation use cases in Northern Sweden. Each Blueprint is reconciled by a specific executor type.

## Equipment and Predictive Maintenance

### VibrationMonitor

Deploys an ML model that analyzes vibration sensor data to predict bearing failures in crushers and conveyors.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: vibration-monitor-
  namespace: mining
spec:
  kind: VibrationMonitor
  data:
    equipment:
      id: "crusher-01"
      type: "jaw-crusher"
      location: "processing-plant-a"
    model:
      name: "vibration-anomaly-detector"
      version: "v2.3.1"
      threshold: 0.85
    sensors:
      - id: "vib-001"
        position: "motor-bearing"
        samplingRate: 1000
      - id: "vib-002"
        position: "drive-shaft"
        samplingRate: 1000
    alerts:
      warning: 0.7
      critical: 0.9
      recipients:
        - "maintenance-team@mine.se"
```

### EquipmentDigitalTwin

Creates a real-time digital twin simulation of a haul truck for maintenance optimization.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: digitaltwin-truck-
  namespace: mining
spec:
  kind: EquipmentDigitalTwin
  data:
    equipment:
      id: "haul-truck-14"
      model: "CAT 797F"
      serialNumber: "HT797F-2024-0014"
    simulation:
      updateInterval: 5
      components:
        - engine
        - transmission
        - hydraulics
        - tires
        - brakes
    dataSources:
      - type: "canbus"
        endpoint: "mqtt://edge-gateway-pit:1883/trucks/14"
      - type: "gps"
        endpoint: "mqtt://edge-gateway-pit:1883/gps/14"
    storage:
      bucket: "digitaltwins"
      retentionDays: 90
```

## Safety and Environmental

### AirQualityMonitor

Processes dust and gas sensors in underground tunnels for worker safety.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: airquality-
  namespace: mining
spec:
  kind: AirQualityMonitor
  data:
    zone:
      id: "tunnel-section-b4"
      type: "underground"
      depth: 450
    sensors:
      dust:
        - id: "dust-b4-01"
          type: "pm2.5"
          threshold: 3.0
        - id: "dust-b4-02"
          type: "pm10"
          threshold: 10.0
      gas:
        - id: "gas-b4-01"
          type: "CO"
          threshold: 25
        - id: "gas-b4-02"
          type: "NO2"
          threshold: 3
        - id: "gas-b4-03"
          type: "methane"
          threshold: 1.0
    ventilation:
      controllerId: "vent-controller-b4"
      autoAdjust: true
    alerts:
      immediate:
        - "control-room@mine.se"
        - "safety-officer@mine.se"
      escalation:
        after: 300
        to: "emergency@mine.se"
```

### GroundStabilityAnalyzer

ML model analyzing seismic and strain sensors to predict rock falls.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: ground-stability-
  namespace: mining
spec:
  kind: GroundStabilityAnalyzer
  data:
    zone:
      id: "open-pit-west-wall"
      type: "slope"
      height: 180
      angle: 45
    model:
      name: "slope-stability-predictor"
      version: "v1.4.0"
    sensors:
      seismic:
        - id: "seis-w01"
          depth: 50
        - id: "seis-w02"
          depth: 100
        - id: "seis-w03"
          depth: 150
      strain:
        - id: "strain-w01"
          type: "extensometer"
        - id: "strain-w02"
          type: "inclinometer"
      radar:
        id: "ssr-west-01"
        type: "slope-stability-radar"
    thresholds:
      displacement: 5.0
      velocity: 0.5
      acceleration: 0.01
    evacuationZone:
      radius: 500
      assembly: "muster-point-alpha"
```

## Autonomous Operations

### AutonomousHaulRoute

Path planning and route optimization for autonomous haul trucks.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: haul-route-
  namespace: mining
spec:
  kind: AutonomousHaulRoute
  data:
    route:
      id: "pit-to-crusher-primary"
      origin:
        zone: "loading-zone-3"
        coordinates: [67.8523, 20.2251]
      destination:
        zone: "crusher-dump"
        coordinates: [67.8612, 20.2189]
    constraints:
      maxGrade: 10
      maxSpeed: 40
      minTurnRadius: 25
      avoidZones:
        - "blast-zone-active"
        - "maintenance-area"
    optimization:
      objective: "fuel-efficiency"
      replanInterval: 300
    vehicles:
      - "haul-truck-12"
      - "haul-truck-14"
      - "haul-truck-17"
    weather:
      adaptToConditions: true
      suspendOnBlizzard: true
```

### DroneInspection

Scheduled drone flights for pit wall inspection and stockpile measurement.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: drone-inspection-
  namespace: mining
spec:
  kind: DroneInspection
  data:
    mission:
      id: "weekly-pit-inspection"
      type: "survey"
      schedule: "0 6 * * 1"
    drone:
      id: "drone-survey-02"
      model: "DJI M300 RTK"
    flightPlan:
      altitude: 120
      overlap: 75
      sidelap: 65
      area:
        type: "polygon"
        coordinates:
          - [67.8501, 20.2200]
          - [67.8550, 20.2200]
          - [67.8550, 20.2300]
          - [67.8501, 20.2300]
    sensors:
      - type: "rgb"
        resolution: "20mp"
      - type: "lidar"
        pointsPerSecond: 240000
      - type: "thermal"
        resolution: "640x512"
    processing:
      outputs:
        - "orthomosaic"
        - "dem"
        - "pointcloud"
      storage: "s3://survey-data/pit-inspections"
    weather:
      maxWind: 12
      minVisibility: 5000
      noRain: true
```

## Edge and Connectivity

### EdgeGateway

Deploys an edge executor at a remote site with offline support.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: edge-gateway-
  namespace: mining
spec:
  kind: EdgeGateway
  data:
    location:
      id: "remote-pit-north"
      coordinates: [68.1234, 20.5678]
      connectivity: "satellite"
    hardware:
      type: "industrial-pc"
      model: "Advantech ARK-3530"
    executor:
      image: "colonyos/edge-executor:v1.2.0"
      resources:
        cpu: "4000m"
        memory: "8Gi"
        storage: "500Gi"
    connectivity:
      primary: "starlink"
      backup: "lte"
      syncInterval: 300
    offlineMode:
      enabled: true
      bufferSize: "100Gi"
      priorityJobs:
        - "safety-alerts"
        - "equipment-critical"
    localServices:
      - name: "mqtt-broker"
        port: 1883
      - name: "timeseries-db"
        port: 8086
```

### SensorAggregator

Collects and aggregates data from a mesh network of sensors.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: sensor-aggregator-
  namespace: mining
spec:
  kind: SensorAggregator
  data:
    zone:
      id: "processing-plant"
      type: "industrial"
    protocols:
      - type: "modbus-tcp"
        port: 502
      - type: "mqtt"
        broker: "mqtt://localhost:1883"
      - type: "opcua"
        endpoint: "opc.tcp://plc-gateway:4840"
    sensors:
      count: 847
      types:
        - temperature
        - pressure
        - flow
        - level
        - vibration
    aggregation:
      interval: 10
      method: "average"
      outlierDetection: true
    output:
      format: "parquet"
      destination: "s3://sensor-data/processing-plant"
      partitionBy: "hour"
```

## Production and Logistics

### OreGradeEstimator

ML model estimating ore grade from drill and blast hole samples.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: ore-grade-
  namespace: mining
spec:
  kind: OreGradeEstimator
  data:
    block:
      id: "block-2024-w03"
      bench: "1240"
      coordinates:
        min: [67.8510, 20.2210]
        max: [67.8520, 20.2220]
    model:
      name: "grade-estimation-rf"
      version: "v3.1.0"
      minerals:
        - name: "iron"
          unit: "percent"
        - name: "phosphorus"
          unit: "ppm"
        - name: "silica"
          unit: "percent"
    samples:
      source: "blastholes"
      interpolation: "kriging"
      variogram: "spherical"
    output:
      blockModel: "s3://geology/block-models/2024-w03"
      gradePlan: "s3://planning/grade-control/2024-w03"
    integration:
      dispatchSystem: "modular-mining"
      crusherOptimizer: true
```

### CrusherOptimizer

Controls crusher settings based on ore hardness and feed characteristics.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: crusher-optimizer-
  namespace: mining
spec:
  kind: CrusherOptimizer
  data:
    equipment:
      id: "primary-crusher-01"
      type: "gyratory"
      model: "FLSmidth TS"
    control:
      plcAddress: "192.168.10.50"
      protocol: "modbus-tcp"
    optimization:
      objective: "throughput"
      constraints:
        power: 5000
        productSize: 150
    inputs:
      - name: "feedRate"
        source: "weightometer-01"
      - name: "oreHardness"
        source: "grade-estimator"
      - name: "moistureContent"
        source: "moisture-sensor-01"
    outputs:
      - name: "css"
        range: [150, 200]
      - name: "eccSpeed"
        range: [280, 320]
    safety:
      maxPower: 5500
      maxPressure: 280
      emergencyStop: true
```

## Cold Climate Specific

### FreezeProtection

Monitors and controls heating systems to prevent equipment freezing.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: freeze-protection-
  namespace: mining
spec:
  kind: FreezeProtection
  data:
    zone:
      id: "water-treatment-plant"
      type: "critical-infrastructure"
    monitoring:
      sensors:
        - id: "temp-pipe-01"
          location: "inlet-pipe"
          threshold: 2.0
        - id: "temp-pipe-02"
          location: "outlet-pipe"
          threshold: 2.0
        - id: "temp-ambient"
          location: "outdoor"
    heating:
      systems:
        - id: "heat-trace-inlet"
          type: "electric-heat-trace"
          power: 15
        - id: "heat-trace-outlet"
          type: "electric-heat-trace"
          power: 15
        - id: "building-heater"
          type: "oil-furnace"
          power: 50
    control:
      plcAddress: "192.168.10.100"
      mode: "automatic"
      preemptive: true
      forecastSource: "smhi-api"
    alerts:
      warning: 5.0
      critical: 2.0
      recipients:
        - "facilities@mine.se"
```

### WeatherAdaptation

Adjusts mining operations based on weather forecasts and conditions.

```yaml
apiVersion: colony.colonyos.io/v1
kind: Blueprint
metadata:
  generateName: weather-adaptation-
  namespace: mining
spec:
  kind: WeatherAdaptation
  data:
    location:
      coordinates: [67.8523, 20.2251]
      timezone: "Europe/Stockholm"
    sources:
      - type: "forecast"
        provider: "smhi"
        updateInterval: 3600
      - type: "local"
        station: "weather-station-01"
        updateInterval: 60
    rules:
      - condition: "wind > 20"
        action: "suspend-crane-operations"
      - condition: "visibility < 500"
        action: "reduce-vehicle-speed"
        parameter: 20
      - condition: "temperature < -30"
        action: "cold-start-protocol"
      - condition: "snowfall > 10"
        action: "activate-snow-clearing"
      - condition: "blizzard-warning"
        action: "evacuate-open-pit"
    notifications:
      channels:
        - "control-room"
        - "shift-supervisors"
      leadTime: 3600
```

## Reconciler Type Summary

| Blueprint Kind | Reconciler Type | Description |
|---------------|-----------------|-------------|
| VibrationMonitor | ml-executor | ML-based sensor analysis |
| EquipmentDigitalTwin | simulation-executor | Real-time equipment simulation |
| AirQualityMonitor | iot-executor | IoT sensor processing |
| GroundStabilityAnalyzer | ml-executor | ML-based geotechnical analysis |
| AutonomousHaulRoute | vehicle-executor | Autonomous vehicle control |
| DroneInspection | drone-executor | Drone flight operations |
| EdgeGateway | docker-reconciler | Container deployment at edge |
| SensorAggregator | iot-executor | Sensor data collection |
| OreGradeEstimator | ml-executor | ML-based grade estimation |
| CrusherOptimizer | plc-executor | Industrial control systems |
| FreezeProtection | plc-executor | Heating system control |
| WeatherAdaptation | container-executor | Weather-based automation |
