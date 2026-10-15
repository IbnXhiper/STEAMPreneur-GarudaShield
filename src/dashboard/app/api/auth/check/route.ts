import { NextRequest, NextResponse } from "next/server";

// Returns whether the current request carries a valid admin key cookie.
// Used by the login page to skip the form when already authenticated, without
// relying on the public /api/status (which is 200 for everyone and previously
// caused a redirect loop back to /login).
export async function GET(req: NextRequest) {
  const key = req.cookies.get("gs_key")?.value;
  if (!key) return NextResponse.json({ authed: false });
  const up = await fetch(`${process.env.GS_API_URL || "http://localhost:8080"}/api/stats`, {
    headers: { "x-gs-key": key },
    cache: "no-store",
  }).catch(() => null);
  return NextResponse.json({ authed: !!up && up.ok });
}
