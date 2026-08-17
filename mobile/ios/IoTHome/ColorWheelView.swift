import SwiftUI
import UIKit

// MARK: - Цветовое колесо (RGB/HSV) для управления RGB-освещением

struct ColorWheelView: View {
    @Binding var hue: Double         // 0..360
    @Binding var saturation: Double  // 0..1
    @Binding var brightness: Double  // 0..1

    private let ring: CGFloat = 260

    var selectedColor: Color {
        Color(hue: hue / 360, saturation: saturation, brightness: brightness)
    }

    var hex: String {
        let ui = UIColor(selectedColor)
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        ui.getRed(&r, green: &g, blue: &b, alpha: &a)
        return String(format: "%02X%02X%02X", Int(r * 255), Int(g * 255), Int(b * 255))
    }

    var body: some View {
        VStack(spacing: 20) {
            ZStack {
                // кольцо оттенка
                Circle()
                    .strokeBorder(
                        AngularGradient(
                            gradient: Gradient(colors: hueRingColors),
                            center: .center,
                            angle: .degrees(0)
                        ),
                        lineWidth: 40
                    )
                    .frame(width: ring, height: ring)

                // внутренняя область: насыщенность/яркость
                Circle()
                    .fill(
                        RadialGradient(
                            colors: [.white, selectedColor, .black],
                            center: .center,
                            startRadius: 0,
                            endRadius: ring / 2 - 40
                        )
                    )
                    .frame(width: ring - 80, height: ring - 80)

                // маркер оттенка
                Circle()
                    .fill(Color(hue: hue / 360, saturation: 1, brightness: 1))
                    .frame(width: 22, height: 22)
                    .overlay(Circle().stroke(.white, lineWidth: 2))
                    .shadow(radius: 2)
                    .offset(knobOffset)
            }
            .frame(width: ring, height: ring)
            .gesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { value in
                        let center = CGPoint(x: ring / 2, y: ring / 2)
                        let dx = value.location.x - center.x
                        let dy = value.location.y - center.y
                        var angle = atan2(Double(dy), Double(dx)) * 180 / .pi
                        if angle < 0 { angle += 360 }
                        hue = angle
                    }
            )

            VStack(spacing: 12) {
                labeledSlider("Насыщенность", value: $saturation)
                labeledSlider("Яркость", value: $brightness)
            }

            HStack {
                RoundedRectangle(cornerRadius: 8)
                    .fill(selectedColor)
                    .frame(width: 44, height: 44)
                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(.black.opacity(0.1)))
                Text("#\(hex)")
                    .font(.system(.body, design: .monospaced))
                Spacer()
                Text("\(Int(hue))° · \(Int(saturation * 100))% · \(Int(brightness * 100))%")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding()
    }

    private var hueRingColors: [Color] {
        stride(from: 0.0, through: 360.0, by: 30.0).map {
            Color(hue: $0 / 360, saturation: 1, brightness: 1)
        }
    }

    private var knobOffset: CGSize {
        let radius = ring / 2 - 20
        let angle = hue * .pi / 180
        return CGSize(width: radius * cos(angle), height: radius * sin(angle))
    }

    private func labeledSlider(_ label: String, value: Binding<Double>) -> some View {
        HStack {
            Text(label).font(.footnote).frame(width: 110, alignment: .leading)
            Slider(value: value, in: 0...1)
        }
    }
}
