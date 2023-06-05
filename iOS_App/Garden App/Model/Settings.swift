//
//  SettingsModel.swift
//  Garden App
//
//  Created by Calvin McLean on 6/12/21.
//

import Foundation

class Server: ObservableObject {
    var scheme: Scheme
    var address: String
    var port: String
    
    var url: String {
        scheme.displayName + address + ":" + port
    }
    
    enum Scheme: String, Identifiable, CaseIterable {
        case http, https
        var displayName: String { rawValue + "://" }
        var id: String { self.rawValue }
    }
    
    init(scheme: Scheme, address: String, port: String) {
        self.scheme = scheme
        self.address = address
        self.port = port
    }
    
    static let `default` = Server(scheme: Scheme.http, address: "nuc.local", port: "30001")
    
    func asMap() -> [String: String] {
        return [
            "scheme": self.scheme.rawValue,
            "address": self.address,
            "port": self.port
        ]
    }
    
    static func fromMap(_ data: [String: Any]?) -> Server {
        if let data = data {
            return Server(
                scheme: Scheme(rawValue: data["scheme"] as? String ?? Scheme.http.rawValue) ?? Scheme.http,
                address: data["address"] as? String ?? "nuc.local",
                port: data["port"] as? String ?? "30001"
            )
        }
        return .default
    }
}

class UserPreferences: ObservableObject {
    var temperatureUnit: TemperatureUnit
    var rainUnit: RainUnit
    
    static let `default` = UserPreferences(temperatureUnit: TemperatureUnit.fahrenheit, rainUnit: RainUnit.inch)
    
    init(temperatureUnit: TemperatureUnit, rainUnit: RainUnit) {
        self.temperatureUnit = temperatureUnit
        self.rainUnit = rainUnit
    }
    
    func asMap() -> [String: String] {
        return [
            "rain_unit": self.rainUnit.displayName,
            "temperature_unit": self.temperatureUnit.displayName
        ]
    }
    
    static func fromMap(_ data: [String: Any]?) -> UserPreferences {
        if let data = data {
            return UserPreferences(
                temperatureUnit: TemperatureUnit(rawValue: data["temperature_unit"] as? String ?? "°F")!,
                rainUnit: RainUnit(rawValue: data["rain_unit"] as? String ?? "in")!
            )
        }
        return .default
    }
}

class Settings {
    struct Constants {
        static let gardenServerKey = "gardenServer"
        static let userPreferencesKey = "userPreferences"
    }
    
    let defaults: UserDefaults
    var server: Server
    var userPreferences: UserPreferences
    
    static let `default` = Settings()
    
    init() {
        self.defaults = .standard
        self.server = Server.fromMap(self.defaults.dictionary(forKey: Constants.gardenServerKey))
        self.userPreferences = UserPreferences.fromMap(self.defaults.dictionary(forKey: Constants.userPreferencesKey))
    }
    
    func save(newSettings: Settings) {
        self.server = newSettings.server
        self.defaults.setValue(self.server.asMap(), forKey: Constants.gardenServerKey)
        self.defaults.setValue(self.userPreferences.asMap(), forKey: Constants.userPreferencesKey)
    }
}

enum RainUnit: String, Identifiable, CaseIterable {
    var id: String { self.rawValue }
    var displayName: String { rawValue }

    case millimeter = "mm", inch = "in"
}

enum TemperatureUnit: String, Identifiable, CaseIterable {
    var id: String { self.rawValue }
    var displayName: String { rawValue }

    case celsius = "°C", fahrenheit = "°F"
}
