// errorText turns whatever a rejected backend call carries into the message to
// show the user (Error, string, or an object with a message). It never returns
// an empty text: a refused change must always be explained by something.
export function errorText(e: unknown): string {
  let text: string;
  if (e instanceof Error) text = e.message;
  else if (typeof e === "string") text = e;
  else if (e && typeof e === "object" && "message" in e && typeof (e as { message: unknown }).message === "string") {
    text = (e as { message: string }).message;
  } else text = String(e);
  return text.trim() === "" ? "The change was refused." : text;
}
