import type { Metadata } from "next";
import "@/styles/index.css";
import { AuthProvider } from "@/context/AuthContext";

export const metadata: Metadata = {
  title: "StockPulse | Seu Dashboard Financeiro",
  description: "Monitoramento de Ações, FIIs e Criptomoedas em tempo real.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt-BR">
      <body>
        <AuthProvider>
          {children}
        </AuthProvider>
      </body>
    </html>
  );
}
