//
//  SettingsHost.swift
//  Garden App
//
//  Created by Calvin McLean on 6/12/21.
//

import SwiftUI

struct SettingsHost: View {
    @EnvironmentObject var modelData: ModelData
    @State private var draftSettings = Settings.default
    
    var body: some View {
        SettingsEditor(settings: $draftSettings)
            .onAppear {
                draftSettings = modelData.settings
            }
            .onDisappear {
                modelData.settings.save(newSettings: draftSettings)
            }
    }
}
