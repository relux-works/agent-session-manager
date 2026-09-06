--- M-F1a: discover =="" admits empty path ("" -> "\x00")
  verdict=BEHAVIOURAL-KILL behavioural=['TestDiscoverRefusals', 'TestEveryArmWitnessRefusesAtTheProductionEntry', 'TestEveryArmWitnessRefusesAtTheProductionEntry/ctor|failInvalid|discovery_candidate_carries_no_executable_path/pathless_candidate'] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F1b: vocab loop admits "" via widened comparison
  verdict=BEHAVIOURAL-KILL behavioural=['TestVocabularyMembershipRefusesEmpty'] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F1d: error-finding gate admits error (== "error" -> == "err")
  verdict=BEHAVIOURAL-KILL behavioural=['TestEveryArmWitnessRefusesAtTheProductionEntry', 'TestEveryArmWitnessRefusesAtTheProductionEntry/ctor|failProtocol|validate_reports_valid_with_an_error_finding/valid_with_error', 'TestValidateResultRules', 'TestValidateResultRules/valid_with_error_finding'] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F1e: target archive_only admitted (literal -> "archive-only")
  verdict=BEHAVIOURAL-KILL behavioural=['TestCheckTupleAdmissionRefusals', 'TestEveryArmWitnessRefusesAtTheProductionEntry', 'TestEveryArmWitnessRefusesAtTheProductionEntry/ctor|failUnsupportedTuple|tuple_target_entry_admits_archive_only/archived_target'] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F1c-probe: reviewer == plant passes unrostered?
  verdict=CENSUS-ONLY behavioural=[] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F1c-control: reviewer != control still reddens census
  verdict=CENSUS-ONLY behavioural=[] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F2a: reviewer var-alias plant (N8 shape on bound helper)
  verdict=CENSUS-ONLY behavioural=[] census=['TestBoundGuardsAreCensused'] restored=True
--- M-F2b: reviewer return-len plant
  verdict=CENSUS-ONLY behavioural=[] census=['TestBoundGuardsAreCensused'] restored=True
--- M-F2c: reviewer for-len plant
  verdict=CENSUS-ONLY behavioural=[] census=['TestBoundGuardsAreCensused'] restored=True
--- M-F2d: control direct helper call still derives
  verdict=CENSUS-ONLY behavioural=[] census=['TestBoundGuardsAreCensused'] restored=True
--- M-F2e: bound widened behaviourally (cursor max 1024 -> 1025)
  verdict=BEHAVIOURAL-KILL behavioural=['TestRequestScalarBoundEdges', 'TestRequestScalarBoundEdges/discover/cursor'] census=[] restored=True
--- M-F3a: CapabilityUsable admits conditional
  verdict=BEHAVIOURAL-KILL behavioural=['TestCapabilityNonAvailableEnabledIsNotUsable', 'TestCapabilityNonAvailableEnabledIsNotUsable/conditional/enabled'] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F3b: capabilityMapUsable admits conditional
  verdict=BEHAVIOURAL-KILL behavioural=['TestCapabilityNonAvailableEnabledIsNotUsable', 'TestCapabilityNonAvailableEnabledIsNotUsable/conditional/enabled', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_native_read_back', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_native_resume_plan', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_workspace_binding', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_writer_pair'] census=['TestIdentityGatesAreCensused'] restored=True
--- M-F3c: provider side admits conditional (!= available && != conditional)
  verdict=BEHAVIOURAL-KILL behavioural=['TestCheckTargetWriteGatesRefusesNonAvailableEnabled', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_provider'] census=['TestIdentityGatesAreCensused', 'TestNoUnregisteredInlineVocabularies'] restored=True

=== SUMMARY ===
M-F1a: BEHAVIOURAL-KILL behav=['TestDiscoverRefusals', 'TestEveryArmWitnessRefusesAtTheProductionEntry', 'TestEveryArmWitnessRefusesAtTheProductionEntry/ctor|failInvalid|discovery_candidate_carries_no_executable_path/pathless_candidate'] census=['TestIdentityGatesAreCensused'] restored=True
M-F1b: BEHAVIOURAL-KILL behav=['TestVocabularyMembershipRefusesEmpty'] census=['TestIdentityGatesAreCensused'] restored=True
M-F1d: BEHAVIOURAL-KILL behav=['TestEveryArmWitnessRefusesAtTheProductionEntry', 'TestEveryArmWitnessRefusesAtTheProductionEntry/ctor|failProtocol|validate_reports_valid_with_an_error_finding/valid_with_error', 'TestValidateResultRules', 'TestValidateResultRules/valid_with_error_finding'] census=['TestIdentityGatesAreCensused'] restored=True
M-F1e: BEHAVIOURAL-KILL behav=['TestCheckTupleAdmissionRefusals', 'TestEveryArmWitnessRefusesAtTheProductionEntry', 'TestEveryArmWitnessRefusesAtTheProductionEntry/ctor|failUnsupportedTuple|tuple_target_entry_admits_archive_only/archived_target'] census=['TestIdentityGatesAreCensused'] restored=True
M-F1c-probe: CENSUS-ONLY behav=[] census=['TestIdentityGatesAreCensused'] restored=True
M-F1c-control: CENSUS-ONLY behav=[] census=['TestIdentityGatesAreCensused'] restored=True
M-F2a: CENSUS-ONLY behav=[] census=['TestBoundGuardsAreCensused'] restored=True
M-F2b: CENSUS-ONLY behav=[] census=['TestBoundGuardsAreCensused'] restored=True
M-F2c: CENSUS-ONLY behav=[] census=['TestBoundGuardsAreCensused'] restored=True
M-F2d: CENSUS-ONLY behav=[] census=['TestBoundGuardsAreCensused'] restored=True
M-F2e: BEHAVIOURAL-KILL behav=['TestRequestScalarBoundEdges', 'TestRequestScalarBoundEdges/discover/cursor'] census=[] restored=True
M-F3a: BEHAVIOURAL-KILL behav=['TestCapabilityNonAvailableEnabledIsNotUsable', 'TestCapabilityNonAvailableEnabledIsNotUsable/conditional/enabled'] census=['TestIdentityGatesAreCensused'] restored=True
M-F3b: BEHAVIOURAL-KILL behav=['TestCapabilityNonAvailableEnabledIsNotUsable', 'TestCapabilityNonAvailableEnabledIsNotUsable/conditional/enabled', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_native_read_back', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_native_resume_plan', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_workspace_binding', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_writer_pair'] census=['TestIdentityGatesAreCensused'] restored=True
M-F3c: BEHAVIOURAL-KILL behav=['TestCheckTargetWriteGatesRefusesNonAvailableEnabled', 'TestCheckTargetWriteGatesRefusesNonAvailableEnabled/conditional_provider'] census=['TestIdentityGatesAreCensused', 'TestNoUnregisteredInlineVocabularies'] restored=True
SNAPSHOT-OK: True
EXTRA-LEFT: []
