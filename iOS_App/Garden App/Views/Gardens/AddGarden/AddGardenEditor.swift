//
//  AddGardenEditor.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/22.
//

import SwiftUI

struct AddGardenEditor: View {
    @EnvironmentObject var modelData: ModelData
    @State(initialValue: Calendar(identifier: .gregorian).date(from: DateComponents(hour: 22, minute: 0, second: 0)) ?? Date()) var startTime: Date
    
    @Environment(\.dismiss) var dismiss
    
    @Binding var newGarden: CreateGardenRequest
    
    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Garden")) {
                    HStack {
                        Text("Name")
                            .frame(width: 100, alignment: .leading)
                        TextField("", text: $newGarden.name)
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                            .autocapitalization(.words)
                    }
                    HStack {
                        Text("Topic Prefix")
                            .frame(width: 100, alignment: .leading)
                        TextField("", text: $newGarden.topicPrefix)
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                            .autocapitalization(.none)
                    }
                    HStack {
                        Text("Max Zones:")
                        Text("\(newGarden.maxZones)")
                        Stepper("", value: $newGarden.maxZones, in: 0...100)
                    }
                }
                
                EditLightSchedule(lightSchedule: $newGarden.lightSchedule)
                
                Section {
                    Button(action: {
                        modelData.gardenResource().createGarden(garden: newGarden)
                        dismiss()
                    }) {
                        Text("Submit")
                    }
                }
            }
            .navigationBarTitle("Add a Garden")
        }
    }
}

struct EditLightSchedule: View {
    @State var lightScheduleEnabled: Bool = false
    @State var lightDurationHour: Float = 14
    @State(initialValue: Calendar(identifier: .gregorian).date(from: DateComponents(year: 2023, hour: 22, minute: 0, second: 0)) ?? Date()) var startTime: Date
    
    @Binding var lightSchedule: LightSchedule?
    
    var body: some View {
        Section(header: Text("Light Schedule")) {
            Toggle(isOn: $lightScheduleEnabled) {
                Text("Enabled")
            }.onChange(of: lightScheduleEnabled) {value in
                lightSchedule = LightSchedule()
            }
            if (lightScheduleEnabled) {
                VStack {
                    Text("Duration: \(Int(lightDurationHour)) hours").frame(maxWidth: .infinity, alignment: .leading)
                    Slider(value: $lightDurationHour, in: 0...23, step: 1)
                        .onChange(of: lightDurationHour) { _ in
                            lightSchedule?.duration = "\(Int(lightDurationHour))h"
                        }
                        .onAppear {
                            lightSchedule?.duration = "\(Int(lightDurationHour))h"
                        }
                }

                HStack {
                    Text("Start Time")
                        .frame(width: 100, alignment: .leading)
                    DatePicker("", selection: $startTime, displayedComponents: .hourAndMinute)
                        .labelsHidden()
                        .onChange(of: startTime, perform: { value in
                            lightSchedule?.startTime = startTime.timeFormatted
                        })
                        .onAppear{
                            lightSchedule?.startTime = startTime.timeFormatted
                        }
                }
            }
        }
    }
}
