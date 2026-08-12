import React, { useEffect } from 'react';
import { SearchResult } from './types';
import { usePortfolio } from '@/context/PortfolioContext';

import NewPortfolioModal from './modals/NewPortfolioModal';
import TransactionModal from './modals/TransactionModal';
import FixedIncomeModal from './modals/FixedIncomeModal';
import EditFixedIncomeModal from './modals/EditFixedIncomeModal';

export interface ModalsProps {
  // Portfolio Modal
  showPortfolioModal?: boolean;
  setShowPortfolioModal?: (s: boolean) => void;
  newPortfolioName?: string;
  setNewPortfolioName?: (s: string) => void;
  newPortfolioCurrency?: string;
  setNewPortfolioCurrency?: (s: string) => void;
  isCreatingPortfolio?: boolean;
  handleCreatePortfolio?: (e: React.FormEvent) => void;

  // Transaction Modal
  showTxModal?: boolean;
  setShowTxModal?: (s: boolean) => void;
  editingTxId?: string | null;
  setEditingTxId?: (id: string | null) => void;
  txTicker?: string;
  searchQuery?: string;
  setSearchQuery?: (s: string) => void;
  isSearching?: boolean;
  showDropdown?: boolean;
  searchResults?: SearchResult[];
  handleSelectAsset?: (s: string) => void;
  isAddingTx?: boolean;
  txType?: 'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS';
  setTxType?: (t: 'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS') => void;
  txQuantity?: string | number;
  setTxQuantity?: (q: string | number) => void;
  txUnitPrice?: string | number;
  setTxUnitPrice?: (p: string | number) => void;
  txFee?: string | number;
  setTxFee?: (f: string | number) => void;
  txExchangeRate?: string | number;
  setTxExchangeRate?: (r: string | number) => void;
  txExecutedAt?: string;
  setTxExecutedAt?: (d: string) => void;
  selectedAssetCurrency?: string;
  kpiCurrency?: string;
  handleAddTransaction?: (e: React.FormEvent) => void;

  // Fixed Income Modal
  showFIModal?: boolean;
  setShowFIModal?: (s: boolean) => void;
  showFIEditModal?: boolean;
  setShowFIEditModal?: (s: boolean) => void;
  fiEditTxAssetName?: string;
  setFiEditTxAssetName?: (s: string) => void;
  handleUpdateFITransaction?: (e: React.FormEvent) => void;
  fiInstitution?: string;
  setFiInstitution?: (s: string) => void;
  fiType?: string;
  setFiType?: (s: string) => void;
  fiDebtType?: string;
  setFiDebtType?: (s: string) => void;
  fiIndexer?: string;
  setFiIndexer?: (s: string) => void;
  fiRate?: string | number;
  setFiRate?: (s: string | number) => void;
  fiTxType?: string;
  setFiTxType?: (s: string) => void;
  fiAmount?: string | number;
  setFiAmount?: (s: string | number) => void;
  fiApplicationDate?: string;
  setFiApplicationDate?: (s: string) => void;
  fiMaturityDate?: string;
  setFiMaturityDate?: (s: string) => void;
  isAddingFI?: boolean;
  handleAddFixedIncome?: (e: React.FormEvent) => void;
}

export default function Modals(props: ModalsProps) {
  let context: ReturnType<typeof usePortfolio> | undefined;
  try {
    context = usePortfolio();
  } catch (e) {
    context = undefined;
  }

  // Resolve values from props first, then fallback to PortfolioContext
  const showPortfolioModal = props.showPortfolioModal ?? context?.showPortfolioModal ?? false;
  const setShowPortfolioModal = props.setShowPortfolioModal ?? context?.setShowPortfolioModal ?? (() => {});
  const newPortfolioName = props.newPortfolioName ?? context?.newPortfolioName ?? '';
  const setNewPortfolioName = props.setNewPortfolioName ?? context?.setNewPortfolioName ?? (() => {});
  const newPortfolioCurrency = props.newPortfolioCurrency ?? context?.newPortfolioCurrency ?? 'BRL';
  const setNewPortfolioCurrency = props.setNewPortfolioCurrency ?? context?.setNewPortfolioCurrency ?? (() => {});
  const isCreatingPortfolio = props.isCreatingPortfolio ?? context?.isCreatingPortfolio ?? false;
  const handleCreatePortfolio = props.handleCreatePortfolio ?? context?.handleCreatePortfolio ?? (() => {});

  const showTxModal = props.showTxModal ?? context?.showTxModal ?? false;
  const setShowTxModal = props.setShowTxModal ?? context?.setShowTxModal ?? (() => {});
  const editingTxId = props.editingTxId !== undefined ? props.editingTxId : (context?.editingTxId ?? null);
  const setEditingTxId = props.setEditingTxId ?? context?.setEditingTxId ?? (() => {});
  const txTicker = props.txTicker ?? context?.txTicker ?? '';
  const searchQuery = props.searchQuery ?? context?.searchQuery ?? '';
  const setSearchQuery = props.setSearchQuery ?? context?.setSearchQuery ?? (() => {});
  const isSearching = props.isSearching ?? context?.isSearching ?? false;
  const showDropdown = props.showDropdown ?? context?.showDropdown ?? false;
  const searchResults = props.searchResults ?? context?.searchResults ?? [];
  const handleSelectAsset = props.handleSelectAsset ?? context?.handleSelectAsset ?? (() => {});
  const isAddingTx = props.isAddingTx ?? context?.isAddingTx ?? false;
  const txType = props.txType ?? context?.txType ?? 'BUY';
  const setTxType = props.setTxType ?? context?.setTxType ?? (() => {});
  const txQuantity = props.txQuantity ?? context?.txQuantity ?? '';
  const setTxQuantity = props.setTxQuantity ?? context?.setTxQuantity ?? (() => {});
  const txUnitPrice = props.txUnitPrice ?? context?.txUnitPrice ?? '';
  const setTxUnitPrice = props.setTxUnitPrice ?? context?.setTxUnitPrice ?? (() => {});
  const txFee = props.txFee ?? context?.txFee ?? '';
  const setTxFee = props.setTxFee ?? context?.setTxFee ?? (() => {});
  const txExchangeRate = props.txExchangeRate ?? context?.txExchangeRate ?? 1.0;
  const setTxExchangeRate = props.setTxExchangeRate ?? context?.setTxExchangeRate ?? (() => {});
  const txExecutedAt = props.txExecutedAt ?? context?.txExecutedAt ?? '';
  const setTxExecutedAt = props.setTxExecutedAt ?? context?.setTxExecutedAt ?? (() => {});
  const selectedAssetCurrency = props.selectedAssetCurrency ?? context?.selectedAssetCurrency ?? 'BRL';
  const kpiCurrency = props.kpiCurrency ?? context?.kpiCurrency ?? 'BRL';
  const handleAddTransaction = props.handleAddTransaction ?? context?.handleAddTransaction ?? (() => {});

  const showFIModal = props.showFIModal ?? context?.showFIModal ?? false;
  const setShowFIModal = props.setShowFIModal ?? context?.setShowFIModal ?? (() => {});
  const showFIEditModal = props.showFIEditModal ?? context?.showFIEditModal ?? false;
  const setShowFIEditModal = props.setShowFIEditModal ?? context?.setShowFIEditModal ?? (() => {});
  const fiEditTxAssetName = props.fiEditTxAssetName ?? context?.fiEditTxAssetName ?? '';
  const handleUpdateFITransaction = props.handleUpdateFITransaction ?? context?.handleUpdateFITransaction ?? (() => {});
  const fiInstitution = props.fiInstitution ?? context?.fiInstitution ?? '';
  const setFiInstitution = props.setFiInstitution ?? context?.setFiInstitution ?? (() => {});
  const fiType = props.fiType ?? context?.fiType ?? 'CDB';
  const setFiType = props.setFiType ?? context?.setFiType ?? (() => {});
  const fiDebtType = props.fiDebtType ?? context?.fiDebtType ?? 'POS';
  const setFiDebtType = props.setFiDebtType ?? context?.setFiDebtType ?? (() => {});
  const fiIndexer = props.fiIndexer ?? context?.fiIndexer ?? 'CDI';
  const setFiIndexer = props.setFiIndexer ?? context?.setFiIndexer ?? (() => {});
  const fiRate = props.fiRate ?? context?.fiRate ?? '';
  const setFiRate = props.setFiRate ?? context?.setFiRate ?? (() => {});
  const fiTxType = props.fiTxType ?? context?.fiTxType ?? 'SUBSCRIPTION';
  const setFiTxType = props.setFiTxType ?? context?.setFiTxType ?? (() => {});
  const fiAmount = props.fiAmount ?? context?.fiAmount ?? '';
  const setFiAmount = props.setFiAmount ?? context?.setFiAmount ?? (() => {});
  const fiApplicationDate = props.fiApplicationDate ?? context?.fiApplicationDate ?? '';
  const setFiApplicationDate = props.setFiApplicationDate ?? context?.setFiApplicationDate ?? (() => {});
  const fiMaturityDate = props.fiMaturityDate ?? context?.fiMaturityDate ?? '';
  const setFiMaturityDate = props.setFiMaturityDate ?? context?.setFiMaturityDate ?? (() => {});
  const isAddingFI = props.isAddingFI ?? context?.isAddingFI ?? false;
  const handleAddFixedIncome = props.handleAddFixedIncome ?? context?.handleAddFixedIncome ?? (() => {});

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (showPortfolioModal) setShowPortfolioModal(false);
        if (showTxModal) setShowTxModal(false);
        if (showFIModal) setShowFIModal(false);
        if (showFIEditModal) setShowFIEditModal(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [
    showPortfolioModal,
    showTxModal,
    showFIModal,
    showFIEditModal,
    setShowPortfolioModal,
    setShowTxModal,
    setShowFIModal,
    setShowFIEditModal,
  ]);

  return (
    <>
      <NewPortfolioModal
        showPortfolioModal={showPortfolioModal}
        setShowPortfolioModal={setShowPortfolioModal}
        newPortfolioName={newPortfolioName}
        setNewPortfolioName={setNewPortfolioName}
        newPortfolioCurrency={newPortfolioCurrency}
        setNewPortfolioCurrency={setNewPortfolioCurrency}
        isCreatingPortfolio={isCreatingPortfolio}
        handleCreatePortfolio={handleCreatePortfolio}
      />

      <TransactionModal
        showTxModal={showTxModal}
        setShowTxModal={setShowTxModal}
        editingTxId={editingTxId}
        setEditingTxId={setEditingTxId}
        txTicker={txTicker}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        isSearching={isSearching}
        showDropdown={showDropdown}
        searchResults={searchResults}
        handleSelectAsset={handleSelectAsset}
        isAddingTx={isAddingTx}
        txType={txType}
        setTxType={setTxType}
        txQuantity={txQuantity}
        setTxQuantity={setTxQuantity}
        txUnitPrice={txUnitPrice}
        setTxUnitPrice={setTxUnitPrice}
        txFee={txFee}
        setTxFee={setTxFee}
        txExchangeRate={txExchangeRate}
        setTxExchangeRate={setTxExchangeRate}
        txExecutedAt={txExecutedAt}
        setTxExecutedAt={setTxExecutedAt}
        selectedAssetCurrency={selectedAssetCurrency}
        kpiCurrency={kpiCurrency}
        handleAddTransaction={handleAddTransaction}
      />

      <FixedIncomeModal
        showFIModal={showFIModal}
        setShowFIModal={setShowFIModal}
        fiInstitution={fiInstitution}
        setFiInstitution={setFiInstitution}
        fiType={fiType}
        setFiType={setFiType}
        fiDebtType={fiDebtType}
        setFiDebtType={setFiDebtType}
        fiIndexer={fiIndexer}
        setFiIndexer={setFiIndexer}
        fiRate={fiRate}
        setFiRate={setFiRate}
        fiAmount={fiAmount}
        setFiAmount={setFiAmount}
        fiApplicationDate={fiApplicationDate}
        setFiApplicationDate={setFiApplicationDate}
        fiMaturityDate={fiMaturityDate}
        setFiMaturityDate={setFiMaturityDate}
        isAddingFI={isAddingFI}
        handleAddFixedIncome={handleAddFixedIncome}
      />

      <EditFixedIncomeModal
        showFIEditModal={showFIEditModal}
        setShowFIEditModal={setShowFIEditModal}
        fiEditTxAssetName={fiEditTxAssetName}
        fiTxType={fiTxType}
        setFiTxType={setFiTxType}
        fiAmount={fiAmount}
        setFiAmount={setFiAmount}
        fiApplicationDate={fiApplicationDate}
        setFiApplicationDate={setFiApplicationDate}
        fiMaturityDate={fiMaturityDate}
        setFiMaturityDate={setFiMaturityDate}
        isAddingFI={isAddingFI}
        handleUpdateFITransaction={handleUpdateFITransaction}
      />
    </>
  );
}
