import { useCallback, useEffect, useRef, useState } from "react";
import type { ComputerClient } from "./client";
import { computerReadinessError } from "./readiness";
import type { ComputerActionReceipt, ComputerCapabilities, ComputerObservation, ComputerSessionSnapshot, StartComputerSessionInput } from "./types";

type ComputerState = {
  available: boolean;
  capabilities: ComputerCapabilities | null;
  session: ComputerSessionSnapshot | null;
  observation: ComputerObservation | null;
  receipts: ComputerActionReceipt[];
  loading: boolean;
  error: string | null;
  controlIntent: "pause" | "stop" | null;
};
const initialState: ComputerState = { available: false, capabilities: null, session: null, observation: null, receipts: [], loading: false, error: null, controlIntent: null };

function mergeSnapshot(current: ComputerState, snapshot: ComputerSessionSnapshot): ComputerState {
  const receipt = snapshot.last_receipt;
  const receipts = receipt ? [...current.receipts.filter((item) => item.action_id !== receipt.action_id), receipt] : current.receipts;
  // A snapshot contains metadata only. Preserve image bytes only for the same observation.
  const observation = snapshot.observation?.id === current.observation?.id
    ? current.observation : snapshot.observation ?? current.observation;
  return { ...current, session: snapshot, capabilities: snapshot.capabilities, observation, receipts };
}

export function useComputerSession(client: ComputerClient | null) {
  const [state, setState] = useState<ComputerState>(initialState);
  const current = useRef(initialState);
  const mounted = useRef(false);
  // Every request has one generation; safety controls supersede all pending work.
  const generation = useRef(0);
  const update = useCallback((next: ComputerState) => {
    current.current = next;
    if (mounted.current) setState(next);
  }, []);

  useEffect(() => {
    mounted.current = true;
    ++generation.current;
    update(initialState);
    return () => { mounted.current = false; ++generation.current; };
  }, [client, update]);

  const fail = useCallback((message: string): never => {
    update({ ...current.current, error: message });
    throw new Error(message);
  }, [update]);

  const run = useCallback(async <T,>(operation: () => Promise<T>, apply: (value: T, state: ComputerState) => ComputerState): Promise<T | null> => {
    const version = ++generation.current;
    update({ ...current.current, loading: true, error: null });
    try {
      const value = await operation();
      if (!mounted.current || version !== generation.current) return null;
      update({ ...apply(value, current.current), loading: false });
      return value;
    } catch (error) {
      if (!mounted.current || version !== generation.current) return null;
      update({ ...current.current, loading: false, error: error instanceof Error ? error.message : String(error) });
      throw error;
    }
  }, [update]);

  const loadCapabilities = useCallback(async () => {
    if (!client) return null;
    // Never let a readiness refresh supersede a session safety control.
    if (current.current.session || current.current.loading) return null;
    return run(() => client.getCapabilities(), (value, state) => ({
      ...state, available: value.available, capabilities: value.capabilities,
      error: value.error_message || value.error_code || (!value.available ? "Computer Use is unavailable on this host." : null)
    }));
  }, [client, run]);

  const start = useCallback(async (input: StartComputerSessionInput = { approved: false }) => {
    const state = current.current;
    if (!client) return fail("Computer Use is unavailable");
    if (!input.approved) return fail("Explicit session approval is required");
    const readiness = computerReadinessError(state.available, state.capabilities);
    if (readiness) return fail(readiness);
    if (state.loading || (state.session && state.session.state !== "stopped")) return fail("Stop the current computer session before starting another");
    update({ ...state, session: null, observation: null, receipts: [], controlIntent: null });
    return run(() => client.start(input), (snapshot, state) => mergeSnapshot(state, snapshot));
  }, [client, fail, run, update]);

  const observe = useCallback(async () => {
    const state = current.current;
    const session = state.session;
    if (!client || !session) return fail("No computer session is active");
    if (state.controlIntent || !["ready", "needs_observation"].includes(session.state)) return fail("The computer session cannot observe until it is resumed");
    if (state.loading) return fail("A computer request is already pending");
    return run(() => client.observe(session.session_id), (value, state) => {
      if (value.observation.session_id !== session.session_id) throw new Error("Observation session mismatch");
      const observation = { ...value.observation, image_data: value.image_data, media_type: value.media_type };
      return { ...state, observation, capabilities: observation.capabilities, session: { ...session, state: "ready", observation, capabilities: observation.capabilities } };
    });
  }, [client, fail, run]);

  const control = useCallback(async (kind: "pause" | "resume" | "stop") => {
    const state = current.current;
    const session = state.session;
    if (!client || !session) return fail("No computer session is active");
    if (session.state === "stopped") return fail("The computer session is stopped");
    if (kind !== "stop" && state.controlIntent === "stop") return fail("Stop requested; retry Stop if it failed before continuing");
    if (kind === "resume" && (state.loading || session.state !== "paused")) return fail("Only a confirmed paused session can resume");
    if (kind === "resume") {
      const readiness = computerReadinessError(state.available, state.capabilities);
      if (readiness) return fail(readiness);
    }
    // Retain the safety intent on rejection: outcome is not confirmed; allow Stop retry.
    update({ ...state, controlIntent: kind === "resume" ? state.controlIntent : kind });
    return run(() => client[kind](session.session_id), (snapshot, state) => {
      if (snapshot.session_id !== session.session_id) throw new Error("Control session mismatch");
      if (kind === "stop" && snapshot.state !== "stopped") throw new Error("Backend did not confirm Stop");
      if (kind === "pause" && snapshot.state !== "paused") throw new Error("Backend did not confirm Pause");
      return { ...mergeSnapshot(state, snapshot), controlIntent: null };
    });
  }, [client, fail, run, update]);
  const pause = useCallback(() => control("pause"), [control]);
  const resume = useCallback(() => control("resume"), [control]);
  const stop = useCallback(() => control("stop"), [control]);

  const getReceipt = useCallback(async (actionID: string) => {
    const state = current.current;
    if (!client || !state.session) return fail("No computer session is active");
    if (state.loading || state.controlIntent) return fail("A computer control request is pending");
    const sessionID = state.session.session_id;
    return run(() => client.getReceipt(sessionID, actionID), (receipt, state) => {
      if (receipt.session_id !== sessionID || receipt.action_id !== actionID) throw new Error("Receipt ownership mismatch");
      return { ...state, receipts: [...state.receipts.filter((item) => item.action_id !== receipt.action_id), receipt] };
    });
  }, [client, fail, run]);

  return { ...state, loadCapabilities, start, observe, pause, resume, stop, getReceipt };
}
