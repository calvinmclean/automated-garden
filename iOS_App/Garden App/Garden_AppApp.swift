//
//  Garden_AppApp.swift
//  Garden App
//
//  Created by Calvin McLean on 4/20/21.
//

import SwiftUI

@main
struct Garden_AppApp: App {
    @StateObject private var modelData = ModelData()
    @Environment(\.colorScheme) var colorScheme
    
    init() {
        let navigationBarAppearace = UINavigationBarAppearance()
        navigationBarAppearace.backgroundColor = UIColor(named: "NavigationBarColor")
        
        UINavigationBar.appearance().standardAppearance = navigationBarAppearace
        UINavigationBar.appearance().compactAppearance = navigationBarAppearace
        UINavigationBar.appearance().scrollEdgeAppearance = navigationBarAppearace
    }

    var body: some Scene {
        WindowGroup {
            GardensList()
            .environmentObject(modelData)
            .background(Color.green)
        }
    }
}
