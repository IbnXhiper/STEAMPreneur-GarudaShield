import { NextRequest, NextResponse } from "next/server";

// eslint-disable-next-line @typescript-eslint/no-var-requires
import brain from "@/lib/brain";

export async function POST(req: NextRequest) {
  const { message } = await req.json().catch(() => ({ message: "" }));
  const text = typeof message === "string" ? message : "";
  try {
    const res = (brain as any).respond(text);
    return NextResponse.json(res);
  } catch {
    return NextResponse.json({
      answer:
        "Maaf, terjadi kesalahan internal pada asisten. Coba lagi atau hubungi tim kami via WhatsApp +62 831-8781-5456.",
      chips: ["What is GarudaShield?", "Contact sales"],
      topic: "error",
    });
  }
}
