import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createComputerClient, type ComputerClient } from "./client";
import type { ComputerActionReceipt, ComputerCapabilities, ComputerObservation, ComputerObservationResponse, ComputerSessionSnapshot, StartComputerSessionInput } from "./types";

type ComputerState = {
  capabilities: ComputerCapabilities | null;
  session: ComputerSessionSnapshot | null;
  observation: ComputerObservation | null;
  receipts: ComputerActionReceipt[];
  loading: boolean;
  error: string | null;
};

const initialState: ComputerState = { capabilities: null, session: null, observation: null, receipts: [], loading: false, error: null };

function mergeSnapshot(current: ComputerState, value: ComputerSessionSnapshot | void): ComputerState {
  if (!value) return current;
  const receipts = [...current.receipts, ...(value.receipts ?? []), ...(value.last_receipt ? [value.last_receipt] : [])].filter((item, index, list) => list.findIndex((candidate) => candidate.action_id === item.action_id) === index);
  return { ...current, session: value, capabilities: value.capabilities ?? current.capabilities, observation: value.observation ?? current.observation, receipts };
}

export function useComputerSession(client: ComputerClient | null) {
  const [state, setState] = useState<ComputerState>(initialState);
  const mounted = useRef(true);
  const controlSerial = useRef(0);
  const sessionIDRef = useRef<string | null>(null);
  useEffect(() => () => { mounted.current = false; }, []);
  useEffect(() => { setState(initialState); sessionIDRef.current = null; }, [client]);

  const run = useCallback(async <T,>(operation: () => Promise<T>, update?: (value: T) => void): Promise<T> => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const value = await operation();
      if (mounted.current) {
        setState((current) => ({ ...current, loading: false }));
        update?.(value);
      }
      return value;
    } catch (error) {
      if (mounted.current) setState((current) => ({ ...current, loading: false, error: error instanceof Error ? error.message : String(error) }));
      throw error;
    }
  }, []);

  const loadCapabilities = useCallback(() => {
    if (!client) return Promise.resolve(null);
    return run(() => client.getCapabilities(), (capabilities) => setState((current) => ({ ...current, capabilities })));
  }, [client, run]);
  const start = useCallback((input: StartComputerSessionInput = { approved: false }) => {
    if (!client) return Promise.reject(new Error("Computer Use is unavailable"));
    return run(() => client.start(input), (snapshot) => { sessionIDRef.current = snapshot.session_id; setState((current) => mergeSnapshot(current, snapshot)); });
  }, [client, run]);
  const observe = useCallback(() => {
    const sessionID = state.session?.session_id ?? sessionIDRef.current;
    if (!client || !sessionID) return Promise.reject(new Error("No computer session is active"));
    return run(() => client.observe(sessionID), (value) => setState((current) => {
      if ("image_data" in value || "media_type" in value) {
        const response = value as ComputerObservationResponse;
        const observation = { ...response.observation, image_data: response.image_data, media_type: response.media_type };
        return { ...current, observation, session: { ...current.session, session_id: sessionID, state: current.session?.state ?? "ready", capabilities: observation.capabilities, observation } as ComputerSessionSnapshot, capabilities: observation.capabilities ?? current.capabilities };
      }
      return { ...current, observation: value as unknown as ComputerObservation };
    }));
  }, [client, run, state.session?.session_id]);
  const control = useCallback((operation: (sessionID: string) => Promise<ComputerSessionSnapshot | void>) => {
    const sessionID = state.session?.session_id ?? sessionIDRef.current;
    if (!client || !sessionID) return Promise.reject(new Error("No computer session is active"));
    const serial = ++controlSerial.current;
    return run(() => operation(sessionID), (value) => {
      if (serial === controlSerial.current) setState((current) => mergeSnapshot(current, value));
    });
  }, [client, run, state.session?.session_id]);
  const pause = useCallback(() => client ? control(client.pause) : Promise.reject(new Error("Computer Use is unavailable")), [client, control]);
  const resume = useCallback(() => client ? control(client.resume) : Promise.reject(new Error("Computer Use is unavailable")), [client, control]);
  const stop = useCallback(() => client ? control(client.stop) : Promise.reject(new Error("Computer Use is unavailable")), [client, control]);
  const getReceipt = useCallback((actionID: string) => {
    const sessionID = state.session?.session_id ?? sessionIDRef.current;
    if (!client || !sessionID) return Promise.reject(new Error("No computer session is active"));
    return run(() => client.getReceipt(sessionID, actionID), (receipt) => setState((current) => ({ ...current, receipts: current.receipts.some((item) => item.action_id === receipt.action_id) ? current.receipts : [...current.receipts, receipt] })));
  }, [client, run, state.session?.session_id]);

  const actions = useMemo(() => ({ loadCapabilities, start, observe, pause, resume, stop, getReceipt }), [getReceipt, loadCapabilities, observe, pause, resume, start, stop]);
  return { ...state, available: Boolean(client), ...actions };
}
