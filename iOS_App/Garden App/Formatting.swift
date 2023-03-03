//
//  Formatting.swift
//  Garden App
//
//  Created by Calvin McLean on 6/11/21.
//

import Foundation
import SwiftUI

extension Int {
    var thousandsFormatting: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        let number = self > 1000
            ? NSNumber(value: Float(self) / 1000)
            : NSNumber(value: self)
        return formatter.string(from: number)!
    }
}

extension Date {
    var formatted: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        return formatter.string(from: self)
    }
    
    var formattedWithTime: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .full
        formatter.dateFormat = "yyyy MMM dd HH:mm:ss"
        return formatter.string(from: self)
    }
    
    var formattedWithTimeButNotYear: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .full
        formatter.dateFormat = "dd MMM 'at' HH:mm:ss"
        return formatter.string(from: self)
    }
    
    var minFormatted: String {
        let now = Date()
        // If next watering is today, display relative time
        if Calendar.current.dateComponents([.day], from: self) == Calendar.current.dateComponents([.day], from: now) {
            let formatter = RelativeDateTimeFormatter()
            formatter.unitsStyle = .full
            return formatter.localizedString(for: self, relativeTo: now)
        }
        // Use formattedWithTimeButNotYear if date is more than 7 days in the future
        if self >= Calendar.current.date(byAdding: .day, value: 7, to: now)! {
            return self.formattedWithTimeButNotYear
        }
        let formatter = DateFormatter()
        formatter.dateFormat = "EEEE h:mm a"
        return formatter.string(from: self)
    }
    
    var timeFormatted: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .none
        formatter.dateFormat = "HH:mm:ssXXX"
        return formatter.string(from: self)
    }
    
    var durationFormatted: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .none
        formatter.dateFormat = "HH'h'mm'm'ss's'"
        return formatter.string(from: self)
    }
}

extension Color {
    static var teal: Color {
        Color(UIColor.systemTeal)
    }
}

// https://stackoverflow.com/a/46458771
extension Formatter {
    static let iso8601withFractionalSeconds: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()
    static let iso8601: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()
}

extension JSONDecoder.DateDecodingStrategy {
    static let customISO8601 = custom {
        let container = try $0.singleValueContainer()
        let string = try container.decode(String.self)
        if let date = Formatter.iso8601withFractionalSeconds.date(from: string) ?? Formatter.iso8601.date(from: string) {
            return date
        }
        throw DecodingError.dataCorruptedError(in: container, debugDescription: "Invalid date: \(string)")
    }
}

extension JSONEncoder.DateEncodingStrategy {
    static let customISO8601 = custom {
        var container = $1.singleValueContainer()
        try container.encode(Formatter.iso8601withFractionalSeconds.string(from: $0))
    }
}
