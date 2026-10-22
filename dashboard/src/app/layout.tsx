import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";

const dmSans = localFont({
  src: "../../public/fonts/dm-sans-latin.woff2",
  variable: "--font-dm-sans",
  weight: "400 700",
  display: "swap",
});

const bebasNeue = localFont({
  src: "../../public/fonts/bebas-neue-latin.woff2",
  variable: "--font-bebas-neue",
  weight: "400",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Meteor Dashboard",
  description: "Manage Meteor projects and integrate the Meteor API.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en">
      <body className={`${dmSans.variable} ${bebasNeue.variable}`}>{children}</body>
    </html>
  );
}
