//
//  SettingsEditor.swift
//  Garden App
//
//  Created by Calvin McLean on 6/12/21.
//

import SwiftUI

struct SettingsEditor: View {
    @Binding var settings: Settings
    @Environment(\.dismiss) var dismiss
    
    var body: some View {
        ScrollView {
            VStack {
                HStack {
                    Text("Settings")
                        .bold()
                        .font(.largeTitle)
                    
                    Button(action: { dismiss() }) { Text("Done") }.padding()
                }
                
                Spacer()
                
                Label("Garden Server", systemImage: "externaldrive.badge.icloud")
                    .font(.title2)
                
                Divider()
                
                HStack {
                    Label("Scheme", systemImage: "lock.icloud")
                        .font(.headline.bold())
                        .frame(width: 100, alignment: .leading)
                    Picker("Garden Server Scheme", selection: $settings.server.scheme) {
                        ForEach(Server.Scheme.allCases) { scheme in
                            Text(scheme.displayName).tag(scheme)
                        }
                    }
                    .pickerStyle(SegmentedPickerStyle())
                }
                HStack {
                    Label("Address", systemImage: "network")
                        .font(.headline.bold())
                        .frame(width: 100, alignment: .leading)
                    TextField("Garden Server Address", text: $settings.server.address)
                        .disableAutocorrection(true)
                        .autocapitalization(.none)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                }
                HStack {
                    Label("Port", systemImage: "tray.and.arrow.down")
                        .font(.headline.bold())
                        .frame(width: 100, alignment: .leading)
                    TextField("Garden Server Port", text: $settings.server.port)
                        .keyboardType(.numberPad)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                }
            }
            .padding()
            
            Spacer()
            
            VStack {
                Label("User Preferences", systemImage: "person")
                    .font(.title2)
                
                Divider()
                
                HStack {
                    Label("Temperature Unit", systemImage: "thermometer.sun")
                        .font(.headline.bold())
                        .frame(width: 200, alignment: .leading)
                    Picker("Temperature Unit", selection: $settings.userPreferences.temperatureUnit) {
                        ForEach(TemperatureUnit.allCases) { unit in
                            Text(unit.displayName).tag(unit)
                        }
                    }
                    .pickerStyle(.segmented)
                }
                
                HStack {
                    Label("Rain Unit", systemImage: "ruler")
                        .font(.headline.bold())
                        .frame(width: 200, alignment: .leading)
                    Picker("Rain Unit", selection: $settings.userPreferences.rainUnit) {
                        ForEach(RainUnit.allCases) { unit in
                            Text(unit.displayName).tag(unit)
                        }
                    }
                    .pickerStyle(.segmented)
                }
            }
            .padding()
        }
    }
}
