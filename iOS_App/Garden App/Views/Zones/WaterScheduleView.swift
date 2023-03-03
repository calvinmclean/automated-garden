//
//  WaterScheduleView.swift
//  Garden App
//
//  Created by Calvin McLean on 12/19/22.
//

import SwiftUI

struct WaterScheduleView: View {
    var waterSchedule: WaterSchedule
    
    var body: some View {
        List {
            Section("Water Schedule") {
                DetailHStack(key: "Duration", value: String(waterSchedule.duration))
                DetailHStack(key: "Interval", value: String(waterSchedule.interval))
                DetailHStack(key: "Start Time", value: waterSchedule.startTime.formattedWithTime)
            }
            
            if let weatherControl = waterSchedule.weatherControl {
                if let moistureControl = weatherControl.soilMoisture {
                    Section("Soil Moisture Control") {
                        DetailHStack(key: "Minimum Moisture", value: String(format: "%d%%", moistureControl.minimumMoisture))
                    }
                }
                
                if let rainControl = weatherControl.rain {
                    Section("Rain Control") {
                        ScaleControlDetails(scale: rainControl)
                    }
                }
                
                if let temperatureControl = weatherControl.temperature {
                    Section("Temperature Control") {
                        ScaleControlDetails(scale: temperatureControl)
                    }
                }
            }
        }
    }
}

struct ScaleControlDetails: View {
    @State var scale: ScaleControl

    var body: some View {
        let minValue = scale.baselineValue-scale.range
        let maxValue = scale.baselineValue+scale.range
        let value = minValue + scale.range * 2 * scale.factor
        Gauge(value: value, in: minValue...maxValue) {
        } currentValueLabel: {
            Text("\(String(format: "Factor: %.2f", scale.factor))")
        } minimumValueLabel: {
            Text("\(String(format: "%.2f", minValue))")
        } maximumValueLabel: {
            Text("\(String(format: "%.2f", maxValue))")
        }
        DetailHStack(key: "Baseline Value", value: String(scale.baselineValue))
    }
}
