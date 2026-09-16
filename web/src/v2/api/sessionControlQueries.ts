import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { IdentityConfig } from "../../lib/types";
import type { CreateSessionInput, OperationResult, SendSessionInput, SessionDetail, SessionListFilters, SessionRef } from "../types";
import { sessionControlQueryKeys, useSessionControlClient } from "./sessionControlClient";
import { mergeSessionDetail } from "./sessionDetailMerge";

type CreateSessionMutationInput = Omit<CreateSessionInput, "idempotencyKey"> & { idempotencyKey?: string };
type SendSessionMutationInput = Omit<SendSessionInput, "idempotencyKey"> & { idempotencyKey?: string };
type SessionRefMutationInput = { ref: SessionRef };

export function useSessionList(identity: IdentityConfig, filters: SessionListFilters, enabled = true) {
  const client = useSessionControlClient();
  return useQuery({
    queryKey: sessionControlQueryKeys.list(identity, filters),
    queryFn: ({ signal }) => client.list(identity, filters, signal),
    refetchInterval: client.subscribe ? 5000 : false,
    retry: false,
    enabled
  });
}

export function useSessionDetail(identity: IdentityConfig, ref: SessionRef, enabled = true) {
  const client = useSessionControlClient();
  return useQuery({
    queryKey: sessionControlQueryKeys.detail(identity, ref),
    queryFn: ({ signal }) => client.get(identity, ref, signal),
    structuralSharing: (oldData, newData) => mergeSessionDetail(oldData as SessionDetail | undefined, newData as SessionDetail),
    enabled
  });
}

function useReadback(identity: IdentityConfig) {
  const queryClient = useQueryClient();
  return async (result: OperationResult): Promise<void> => {
    const detailKey = sessionControlQueryKeys.detail(identity, result.session.ref);
    const readback = sessionWithOperationReplay(result.session, result);
    queryClient.setQueryData<SessionDetail>(detailKey, (current) => current?.events ? { ...current, status: readback.status } : readback);
    await queryClient.invalidateQueries({ queryKey: detailKey });
    queryClient.setQueryData<SessionDetail>(detailKey, (current) => current ? sessionWithOperationReplay(current, result) : readback);
    await queryClient.invalidateQueries({ queryKey: ["session-control", "list", identity.tenantKey, identity.userId] });
  };
}

function sessionWithOperationReplay(session: SessionDetail, result: OperationResult): SessionDetail {
  const operationMessage = result.session.messages.find((message) => message.operation?.id === result.operation.id);
  const operationExists = session.messages.some((message) => message.operation?.id === result.operation.id);
  const messages = operationExists ? session.messages : operationMessage ? [...session.messages, operationMessage] : session.messages;
  return {
    ...session,
    messages: messages.map((message) => message.operation?.id === result.operation.id ? {
      ...message,
      operation: { ...message.operation, replayed: result.replayed }
    } : message)
  };
}

export function useCreateSession(identity: IdentityConfig) {
  const client = useSessionControlClient();
  const applyReadback = useReadback(identity);
  return useMutation({
    mutationFn: (input: CreateSessionMutationInput) => client.create(identity, { ...input, idempotencyKey: input.idempotencyKey ?? crypto.randomUUID() }),
    onSuccess: applyReadback
  });
}

export function useSendSession(identity: IdentityConfig) {
  const client = useSessionControlClient();
  const applyReadback = useReadback(identity);
  return useMutation({
    mutationFn: (input: SendSessionMutationInput) => client.send(identity, { ...input, idempotencyKey: input.idempotencyKey ?? crypto.randomUUID() }),
    onSuccess: applyReadback
  });
}

export function useStopSession(identity: IdentityConfig) {
  const client = useSessionControlClient();
  const applyReadback = useReadback(identity);
  return useMutation({
    mutationFn: (input: SessionRefMutationInput) => client.stop(identity, { ...input, idempotencyKey: crypto.randomUUID() }),
    onSuccess: applyReadback
  });
}

export function useArchiveSession(identity: IdentityConfig) {
  const client = useSessionControlClient();
  const applyReadback = useReadback(identity);
  return useMutation({
    mutationFn: (input: SessionRefMutationInput) => client.archive(identity, { ...input, idempotencyKey: crypto.randomUUID() }),
    onSuccess: applyReadback
  });
}
