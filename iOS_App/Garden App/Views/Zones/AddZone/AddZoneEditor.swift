//
//  AddZone.swift
//  Garden App
//
//  Created by Calvin McLean on 6/28/21.
//

import SwiftUI

struct AddZoneEditor: View {
    @EnvironmentObject var modelData: ModelData
    @State var zoneCount: Int = 0
    
    @Environment(\.dismiss) var dismiss
    
    var garden: Garden
    @Binding var newZone: CreateZoneRequest
    
    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Zone")) {
                    HStack {
                        Text("Name").frame(width: 100, alignment: .leading)
                        TextField("", text: $newZone.name)
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                            .autocapitalization(.words)
                    }
                    HStack {
                        Text("Zone Position:")
                        Text("\(newZone.position)")
                        Stepper("", value: $newZone.position, in: 0...garden.maxZones - 1)
                    }
                }
                
                EditWaterSchedule(waterSchedule: $newZone.waterSchedule)
                
                Section(header: Text("Details")) {
                    HStack {
                        Text("Description").frame(width: 100, alignment: .leading)
                        TextField("", text: Binding(get: {newZone.details?.description ?? ""}, set: {newZone.details?.description = $0}))
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                    }
                    HStack {
                        Text("Notes").frame(width: 100, alignment: .leading)
                        TextField("", text: Binding(get: {newZone.details?.notes ?? ""}, set: {newZone.details?.notes = $0}))
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                    }
                }
                
                Section {
                    Button(action: {
                        modelData.zoneResource().createZone(garden: garden, zone: newZone)
                        dismiss()
                    }) {
                        Text("Submit")
                    }
                }
            }
            .navigationBarTitle("Add a Zone")
        }
    }
}

struct EditWaterSchedule: View {
    // TODO: use this if/when API is refactored to only need a time value
    //    @State(initialValue: Calendar(identifier: .gregorian).date(from: DateComponents(hour: 6, minute: 0, second: 0)) ?? Date()) var startTime: Date
    @State(initialValue: Date()) var startTime: Date
    
    @State var waterDurationHour: Int = 0
    @State var waterDurationMinute: Int = 0
    @State var waterDurationInput: Float = 0
    
    @State var waterIntervalDay: Int = 0
    @State var waterIntervalHour: Int = 0
    @State var waterIntervalInput: Float = 0
    
    @Binding var waterSchedule: WaterSchedule
    
    var body: some View {
        Section(header: Text("Water Schedule")) {
            VStack {
                Text("Duration: \(Int(waterDurationHour)) hours \(Int(waterDurationMinute)) minutes")
                    .frame(maxWidth: .infinity, alignment: .leading)

                Slider(value: $waterDurationInput, in: 0...12*60, step: 1)
                    .onChange(of: waterDurationInput) { _ in
                        waterDurationHour = Int(waterDurationInput) / 60
                        waterDurationMinute = Int(waterDurationInput) - waterDurationHour*60
                        waterSchedule.duration = "\(Int(waterDurationHour))h\(Int(waterDurationMinute))m"
                    }
            }
            
            VStack {
                Text("Interval: \(Int(waterIntervalDay)) days \(Int(waterIntervalHour)) hours")
                    .frame(maxWidth: .infinity, alignment: .leading)
                
                Slider(value: $waterIntervalInput, in: 0...24*31, step: 6)
                    .onChange(of: waterIntervalInput) { _ in
                        waterIntervalDay = Int(waterIntervalInput) / 24
                        waterIntervalHour = Int(waterIntervalInput) - waterIntervalDay*24
                        waterSchedule.interval = "\(Int(waterIntervalDay) * 24 + Int(waterIntervalHour))h"
                    }
            }
            
            HStack {
                Text("Start Time").frame(width: 100, alignment: .leading)
                DatePicker("", selection: $startTime, displayedComponents: [.date, .hourAndMinute])
                    .labelsHidden()
                    .onChange(of: startTime) { value in
                        waterSchedule.startTime = startTime
                    }
                    .onAppear{
                        waterSchedule.startTime = startTime
                    }
            }
            EditWeatherControl(weatherControl: $waterSchedule.weatherControl)
        }
    }
}

struct EditWeatherControl: View {
    @State var weatherControlEnabled: Bool = false
    @Binding var weatherControl: WeatherControl?
    
    @State var moistureControlEnabled: Bool = false
    @State var minimumMoisture: Float32 = 0
    
    @State var temperatureControl: ScaleControl? = nil
    @State var rainControl: ScaleControl? = nil
    
    var body: some View {
        Toggle(isOn: $weatherControlEnabled) {
            Text("Enable Weather Control")
        }.onChange(of: weatherControlEnabled) {_ in
            if (weatherControlEnabled) {
                weatherControl = WeatherControl()
            } else {
                weatherControl = nil
            }
        }
        if (weatherControlEnabled) {
            Toggle(isOn: $moistureControlEnabled) {
                Text("Enable Minimum Moisture")
            }.onChange(of: moistureControlEnabled) {_ in
                if (moistureControlEnabled) {
                    minimumMoisture = 50
                    weatherControl?.soilMoisture = SoilMoistureControl()
                } else {
                    weatherControl?.soilMoisture = nil
                }
            }
            if (moistureControlEnabled) {
                VStack {
                    HStack {
                        Text("Minimum Moisture:")
                        Text("\(Int(minimumMoisture))%")
                    }
                    Slider(value: $minimumMoisture, in: 0...100, step: 1)
                        .onChange(of: minimumMoisture) { _ in
                            weatherControl?.soilMoisture?.minimumMoisture = Int(minimumMoisture)
                        }
                }
            }
            EditScaleControl(controlName: "Temperature Control", scaleControl: $temperatureControl)
                .onChange(of: temperatureControl?.factor) { _ in
                    weatherControl?.temperature = temperatureControl
                }
            EditScaleControl(controlName: "Rain Control", scaleControl: $rainControl)
                .onChange(of: rainControl?.factor) { _ in
                    weatherControl?.rain = rainControl
                }
        }
    }
}

struct EditScaleControl: View {
    @State var scaleControlEnabled: Bool = false
    @State var baselineValue: Float32 = 0
    @State var factor: Float32 = 0
    @State var range: Float32 = 0
    
    var controlName: String
    @Binding var scaleControl: ScaleControl?
    
    var body: some View {
        Toggle(isOn: $scaleControlEnabled) {
            Text("Enable \(controlName)")
        }.onChange(of: scaleControlEnabled) {_ in
            if (scaleControlEnabled) {
                scaleControl = ScaleControl()
            } else {
                scaleControl = nil
            }
        }
        if (scaleControlEnabled) {
            HStack {
                Text("Baseline Value:").frame(width: 100, alignment: .leading)
                TextField("", value: $baselineValue, formatter: NumberFormatter())
                    .onChange(of: baselineValue) { _ in
                        scaleControl?.baselineValue = baselineValue
                    }.keyboardType(.decimalPad)
            }
            
            HStack {
                Text("Range:").frame(width: 100, alignment: .leading)
                TextField("", value: $range, formatter: NumberFormatter())
                    .onChange(of: range) { _ in
                        scaleControl?.range = range
                    }.keyboardType(.decimalPad)
            }
            
            let formattedFloat = String(format: "%.2f", factor)
            HStack {
                Text("Factor: \(formattedFloat)")
                Slider(value: $factor, in: 0...1.00, step: 0.01)
                    .onChange(of: factor) { _ in
                        scaleControl?.factor = factor
                    }
            }
        }
    }
}
